package routes

import (
	"encoding/json"
	"game-backend/controller"
	"game-backend/models"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

// Config agrupa opções de configuração das rotas.
type Config struct {
	AllowedOrigins string // Origens permitidas no WebSocket, separadas por vírgula
}

// upgrader global é substituído em InitRoutes por newUpgrader com a lista de
// origens permitidas. O valor inicial usa o comportamento padrão do gorilla
// (verifica se a origem bate com o Host), nunca aceita qualquer origem.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// newUpgrader cria um Upgrader com validação de origem.
func newUpgrader(allowed string) websocket.Upgrader {
	allowedSet := parseOrigins(allowed)

	// Se a lista estiver vazia, não aceita nenhuma origem (mais seguro).
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			// Requisições sem header Origin (ex.: HTTP 1.0 / ferramentas) são negadas
			// a menos que estejam explicitamente permitidas.
			return origin != "" && allowedSet[origin]
		},
	}
}

// parseOrigins converte a lista separada por vírgulas em um conjunto.
func parseOrigins(allowed string) map[string]bool {
	allowedSet := make(map[string]bool)
	for _, origin := range strings.Split(allowed, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			allowedSet[origin] = true
		}
	}
	return allowedSet
}

// withCORS aplica os headers CORS às respostas das rotas de API, ecoando a
// origem da requisição apenas quando ela estiver na lista permitida (mesma
// lista usada na validação do WebSocket, evitando duas listas divergentes).
func withCORS(allowedSet map[string]bool, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowedSet[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func serveWs(hub *controller.Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Erro ao fazer upgrade para WebSocket:", err)
		return
	}

	client := &controller.Client{Hub: hub, Conn: conn, Send: make(chan []byte, 256)}

	hub.Register <- client

	go controller.ReadPump(client)
	go controller.WritePump(client)
}

func getScoresHandler(hub *controller.Hub, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	scores, err := hub.ScoreService.GetTopScores()
	if err != nil {
		log.Printf("Erro ao buscar placares: %v", err)
		http.Error(w, "Erro interno ao buscar placares", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scores)
}

func getConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.GetGameConfig())
}

// InitRoutes configura todas as rotas do seu servidor
func InitRoutes(hub *controller.Hub, cfg Config) {
	allowedSet := parseOrigins(cfg.AllowedOrigins)
	upgrader = newUpgrader(cfg.AllowedOrigins)

	// 1. Rota WebSocket
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	// 2. Rota de API para o Placar
	http.HandleFunc("/api/scores", withCORS(allowedSet, func(w http.ResponseWriter, r *http.Request) {
		getScoresHandler(hub, w, r)
	}))

	// 3. Rota de API para a configuração compartilhada do jogo
	http.HandleFunc("/api/config", withCORS(allowedSet, getConfigHandler))

	log.Printf("Rotas /ws, /api/scores e /api/config configuradas com origens permitidas: %q.", cfg.AllowedOrigins)
}