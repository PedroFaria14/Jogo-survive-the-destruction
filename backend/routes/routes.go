package routes

import (
	"encoding/json"
	"game-backend/controller"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// O upgrader é uma variável privada deste pacote
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Permite conexões de qualquer origem (necessário para o frontend rodando em localhost)
	CheckOrigin: func(r *http.Request) bool { return true },
}

// serveWs configura o novo cliente e inicia as goroutines de I/O
func serveWs(hub *controller.Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Erro ao fazer upgrade para WebSocket:", err)
		return
	}

	// Cria o novo cliente
	client := &controller.Client{Hub: hub, Conn: conn, Send: make(chan []byte, 256)}

	// Registra o cliente no Hub
	hub.Register <- client

	// Chama as funções exportadas (com letra maiúscula) do pacote controller
	go controller.ReadPump(client)
	go controller.WritePump(client)
}

// getScoresHandler lida com a requisição GET /api/scores para buscar o placar.
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

// InitRoutes configura todas as rotas do seu servidor
func InitRoutes(hub *controller.Hub) {
	// 1. Rota WebSocket
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	// 2. Rota de API para o Placar
	http.HandleFunc("/api/scores", func(w http.ResponseWriter, r *http.Request) {
		getScoresHandler(hub, w, r)
	})

	log.Println("Rotas /ws e /api/scores configuradas.")
}
