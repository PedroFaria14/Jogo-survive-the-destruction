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

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// newUpgrader cria um Upgrader com validação de origem.
func newUpgrader(allowed string) websocket.Upgrader {
	allowedSet := make(map[string]bool)
	for _, origin := range strings.Split(allowed, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			allowedSet[origin] = true
		}
	}

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
	upgrader = newUpgrader(cfg.AllowedOrigins)

	// 1. Rota WebSocket
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	// 2. Rota de API para o Placar
	http.HandleFunc("/api/scores", func(w http.ResponseWriter, r *http.Request) {
		getScoresHandler(hub, w, r)
	})

	// 3. Rota de API para a configuração compartilhada do jogo
	http.HandleFunc("/api/config", getConfigHandler)

	log.Printf("Rotas /ws, /api/scores e /api/config configuradas com origens permitidas: %q.", cfg.AllowedOrigins)
}