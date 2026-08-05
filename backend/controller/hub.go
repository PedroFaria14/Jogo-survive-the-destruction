package controller

import (
	"encoding/json"
	"game-backend/models"
	"game-backend/service"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// RoundRestartDelay é o tempo de contagem regressiva antes de iniciar nova rodada.
const RoundRestartDelay = 3 * time.Second

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

	// closed indica que os pumps já encerraram. Evita que um Register chegue
	// depois do Unregister (corrida de desconexão rápida) e crie um player
	// fantasma que nunca seria removido.
	closed atomic.Bool

	// closeOnce garante que o Unregister + fechamento da conexão sejam
	// enviados exatamente uma vez, vindos de qualquer um dos pumps.
	closeOnce sync.Once
}

// Hub gerencia as conexões de clientes e o estado do jogo
type Hub struct {
	Clients      map[*Client]bool
	Register     chan *Client
	Unregister   chan *Client
	Broadcast    chan []byte
	Command      chan PlayerCommand // Canal para comandos de entrada dos jogadores
	GameState    *models.GameState
	ScoreService service.ScoreService // Serviço para lidar com o placar

	// Estado da rodada (serializado apenas via GameState.RoundOver/Countdown)
	roundOver bool
	roundEnd  time.Time
}

// NewHub recebe o serviço de placar para injeção de dependência
func NewHub(ss service.ScoreService) *Hub {
	return &Hub{
		Clients:      make(map[*Client]bool),
		Register:     make(chan *Client),
		Unregister:   make(chan *Client),
		Broadcast:    make(chan []byte),
		Command:      make(chan PlayerCommand),
		GameState:    models.NewGameState(),
		ScoreService: ss,
	}
}

// removeClient remove o cliente do mapa, fecha o canal Send (apenas uma vez)
// e remove o player correspondente do GameState. Deve ser chamado somente a
// partir da goroutine Run() (serializado).
func (h *Hub) removeClient(client *Client) {
	if _, ok := h.Clients[client]; ok {
		delete(h.Clients, client)
		close(client.Send)
	}
	// Remove também do GameState em todos os caminhos (desconexão e eviction
	// de cliente lento), evitando players fantasma acumulados.
	h.GameState.RemovePlayer(client.PlayerID)
}

// broadcastState serializa e envia o estado atual para todos os clientes.
func (h *Hub) broadcastState() {
	stateJSON, err := json.Marshal(h.GameState)
	if err != nil {
		log.Printf("Erro ao serializar GameState: %v", err)
		return
	}

	for client := range h.Clients {
		select {
		case client.Send <- stateJSON:
		default:
			// Cliente lento: remove a conexão para evitar acúmulo.
			h.removeClient(client)
			client.Conn.Close()
		}
	}
}

// saveDeadScores salva (uma única vez) o placar dos jogadores mortos.
// Não fecha a conexão — o jogador vira espectador.
func (h *Hub) saveDeadScores() {
	for _, p := range h.GameState.Players {
		if !p.IsDead || p.ScoreSaved {
			continue
		}
		p.ScoreSaved = true
		duration := time.Since(p.StartTime)
		finalScore := int(duration.Seconds())
		name := p.Name
		if name == "" {
			name = p.ID
		}

		pp := p
		go func() {
			if err := h.ScoreService.SaveScore(pp.ID, name, finalScore, duration); err != nil {
				log.Printf("ERRO ao salvar placar do jogador %s: %v", pp.ID, err)
			}
		}()
	}
}

// roundShouldEnd indica o fim da rodada: arena sem blocos ativos, todos os
// jogadores eliminados, ou (em multiplayer) resta apenas um jogador vivo —
// o último sobrevivente vence a rodada.
func (h *Hub) roundShouldEnd() bool {
	gs := h.GameState
	if gs.NoActiveTiles() {
		return true
	}
	total := 0
	alive := 0
	for _, p := range gs.Players {
		total++
		if !p.IsDead {
			alive++
		}
	}
	if total == 0 {
		return false
	}
	if total >= 2 && alive <= 1 {
		return true
	}
	return alive == 0
}

// tick executa um passo de jogo. Roda dentro da goroutine Run() (serializado).
func (h *Hub) tick(now time.Time) {
	gs := h.GameState

	// 1. Detecta o fim de rodada: todos mortos ou arena sem blocos ativos.
	if !h.roundOver && h.roundShouldEnd() {
		h.roundOver = true
		h.roundEnd = now.Add(RoundRestartDelay)
		gs.RoundOver = true
		gs.Status = "round_over"
		log.Printf("Rodada encerrada. Nova rodada em %s.", RoundRestartDelay)
	}

	// 2. Contagem regressiva / reinício.
	if h.roundOver {
		h.saveDeadScores()

		remaining := int(h.roundEnd.Sub(now).Seconds() + 0.999)
		if remaining < 0 {
			remaining = 0
		}
		gs.Countdown = remaining

		if !now.Before(h.roundEnd) {
			gs.ResetRound()
			h.roundOver = false
			h.roundEnd = time.Time{}
			log.Println("Nova rodada iniciada.")
		}
		h.broadcastState()
		return
	}

	// 3. Rodada ativa: física, power-ups, destruição e placares.
	h.saveDeadScores()
	gs.UpdatePowerUps(now)
	gs.ApplyPhysics()
	gs.CheckArenaDestruction()
	gs.ExpireFallingTiles(now)
	for _, p := range gs.Players {
		if !p.IsDead {
			p.Score = int(now.Sub(p.StartTime).Seconds())
		}
	}

	h.broadcastState()
}

// Run é o único goroutine que lê/escreve o GameState e a lista de clientes.
// Todos os eventos (registro, comandos, desconexão e ticks de física) passam
// por este loop, evitando condições de corrida nos mapas compartilhados.
func (h *Hub) Run() {
	ticker := time.NewTicker(time.Millisecond * 16) // ~60 FPS
	defer ticker.Stop()

	for {
		select {
		case now := <-ticker.C:
			h.tick(now)

		case client := <-h.Register:
			// Conexão pode já ter encerrado (corrida: Unregister consumido
			// antes deste Register). Não cria player fantasma nesse caso.
			if client.closed.Load() {
				client.Conn.Close()
				continue
			}
			h.Clients[client] = true
			log.Printf("Novo cliente conectado. Total de clientes: %d", len(h.Clients))

			// Adiciona o jogador ao estado do jogo e associa o ID ao Client.
			player := h.GameState.AddPlayer(client.Conn.RemoteAddr().String())
			client.PlayerID = player.ID

			// Informa ao cliente qual é o seu próprio ID logo após se conectar.
			initMsg, err := json.Marshal(map[string]string{"type": "init", "player_id": player.ID})
			if err == nil {
				client.Send <- initMsg
			}

		case client := <-h.Unregister:
			h.removeClient(client)
			log.Printf("Cliente desconectado. Total de clientes: %d", len(h.Clients))

		case playerCmd := <-h.Command:
			// Comando de join: define o nickname (sanitizado) e a cor da bolinha.
			if playerCmd.Cmd.Type == "join" {
				name := []rune(strings.TrimSpace(playerCmd.Cmd.Name))
				if len(name) > 20 {
					name = name[:20]
				}
				color := models.SanitizeColor(playerCmd.Cmd.Color)
				if p := h.GameState.Players[playerCmd.PlayerID]; p != nil {
					p.Name = string(name)
					p.Color = color
					log.Printf("Jogador %s definiu o nome: %q e cor: %q", playerCmd.PlayerID, string(name), color)
				}
				continue
			}
			// Comando de restart: respawna o jogador no mesmo mapa com vidas
			// cheias. Apenas jogadores mortos (evita exploit de cura/teleporte).
			if playerCmd.Cmd.Type == "restart" {
				if !h.roundOver {
					if p := h.GameState.Players[playerCmd.PlayerID]; p != nil && p.IsDead {
						p.Lives = models.MaxLives
						h.GameState.RespawnPlayer(playerCmd.PlayerID)
					}
				}
				continue
			}
			// Demais comandos: input de movimento.
			h.GameState.ProcessInput(playerCmd.PlayerID, playerCmd.Cmd)
		}
	}
}
