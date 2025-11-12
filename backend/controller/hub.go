package controller

import (
	"encoding/json"
	"game-backend/models"
	"game-backend/service"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type PlayerCommand struct {
	PlayerID string
	Cmd      models.Command
}

// Client representa a conexão individual de um jogador
type Client struct {
	Conn     *websocket.Conn // Conexão WebSocket
	Send     chan []byte     // Canal para enviar mensagens ao navegador
	Hub      *Hub            // Referência ao Hub
	PlayerID string          // ID do Jogador (chave no mapa GameState)
}

// Hub gerencia as conexões de clientes e o estado do jogo
type Hub struct {
	Clients      map[*Client]bool
	Register     chan *Client
	Unregister   chan *Client
	Broadcast    chan []byte
	Command      chan PlayerCommand // Canal para comandos de entrada dos jogadores
	GameState    *models.GameState
	ScoreService service.ScoreService // NOVO: Serviço para lidar com o placar
}

// NewHub agora recebe o serviço de placar para injeção de dependência
func NewHub(ss service.ScoreService) *Hub {
	return &Hub{
		Clients:      make(map[*Client]bool),
		Register:     make(chan *Client),
		Unregister:   make(chan *Client),
		Broadcast:    make(chan []byte),
		Command:      make(chan PlayerCommand),
		GameState:    models.NewGameState(),
		ScoreService: ss, // ATUALIZADO: Injetado o serviço
	}
}

func (h *Hub) gameLoop() {
	// Rodando a 60 FPS (cerca de 16ms)
	ticker := time.NewTicker(time.Millisecond * 16)
	defer ticker.Stop()

	for range ticker.C {
		// 1. PROCESSAR MORTE e SALVAR PLACAR
		playersToRemove := []*models.Player{}
		for _, player := range h.GameState.Players {
			if player.IsDead {
				// 1.1. Calcular Pontuação Final e Salvar (Não bloqueie o gameLoop!)
				duration := time.Since(player.StartTime)
				finalScore := int(duration.Seconds()) // Placar em segundos

				// Usa goroutine para salvar o placar sem travar o loop do jogo
				go func(p *models.Player, d time.Duration, s int) {
					if err := h.ScoreService.SaveScore(p.ID, s, d); err != nil {
						log.Printf("ERRO ao salvar placar do jogador %s: %v", p.ID, err)
					}
					// 1.2. Após salvar, o jogador é removido do hub para liberar a conexão.
					// Note: Precisamos de uma maneira de encontrar o *Client do PlayerID
					// Esta lógica será delegada ao Unregister no Hub.Run após o cliente se desconectar.
				}(player, duration, finalScore)

				// Adiciona o jogador para ser removido do GameState ao final do loop
				playersToRemove = append(playersToRemove, player)
			} else {
				// 1.3. ATUALIZAR PLACAR (Enquanto vivo, o placar é o tempo de sobrevivência)
				player.Score = int(time.Since(player.StartTime).Seconds())
			}
		}

		// 2. APLICAR FÍSICA E DESTRUIÇÃO
		h.GameState.ApplyPhysics()
		h.GameState.CheckArenaDestruction()

		// 3. SINCRONIZAR (BROADCAST)
		stateJSON, err := json.Marshal(h.GameState)
		if err != nil {
			log.Printf("Erro ao serializar GameState: %v", err)
			continue
		}

		// Envia o estado serializado de volta para o canal de Broadcast
		h.Broadcast <- stateJSON

		// 4. REMOVER CLIENTES MORTOS APÓS O BROADCAST
		// A remoção do GameState acontece aqui, forçando a desconexão do cliente.
		for _, p := range playersToRemove {
			// Encontra o *Client associado ao PlayerID para fechar a conexão
			for client := range h.Clients {
				if client.PlayerID == p.ID {
					// Fecha a conexão do cliente para acionar o Unregister no Hub.Run
					client.Conn.Close()
					break
				}
			}
			h.GameState.RemovePlayer(p.ID)
		}
	}
}

func (h *Hub) Run() {
	// 🛑 INICIA O GAME LOOP EM UMA GOROUTINE SEPARADA!
	go h.gameLoop()

	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = true
			log.Printf("Novo cliente conectado. Total de clientes: %d", len(h.Clients))
			// Adiciona o jogador ao estado do jogo
			player := h.GameState.AddPlayer(client.Conn.RemoteAddr().String())
			client.PlayerID = player.ID // Associa o ID do Player do GameState ao Client

		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
				log.Printf("Cliente desconectado. Total de clientes: %d", len(h.Clients))
				// A remoção do GameState (h.GameState.RemovePlayer) é delegada
				// ao gameLoop quando o jogador morre, ou já acontece aqui
				// se o jogador se desconectar enquanto vivo.
				h.GameState.RemovePlayer(client.PlayerID) // Garante a remoção em qualquer caso
			}
		case message := <-h.Broadcast:
			for client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}
		case playerCmd := <-h.Command:
			// Processa o comando do jogador
			h.GameState.ProcessInput(playerCmd.PlayerID, playerCmd.Cmd)
		}
	}
}
