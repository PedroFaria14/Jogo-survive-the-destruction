package controller

import (
	"context"
	"game-backend/models"
	"game-backend/service"
	"log"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// RoundRestartDelay é o tempo de contagem regressiva antes de iniciar nova rodada.
const RoundRestartDelay = 3 * time.Second

// MaxClients é o teto de conexões WebSocket simultâneas (proteção contra DoS
// por exaustão de recursos em um endpoint público sem autenticação).
const MaxClients = 300

// MinScoreToSave descarta placares triviais (spam de restart em loop),
// protegendo o leaderboard de poluição e o banco de crescimento sem limite.
const MinScoreToSave = 3

type PlayerCommand struct {
	Client *Client
	Cmd    models.Command
}

// Client representa a conexão individual de um jogador
type Client struct {
	Conn     *websocket.Conn // Conexão WebSocket
	Send     chan []byte     // Canal para enviar mensagens ao navegador
	Hub      *Hub            // Referência ao Hub
	PlayerID string          // ID do Jogador (chave no mapa GameState da sala)
	Room     *Room           // Sala (partida) à qual o jogador pertence

	// closed indica que os pumps já encerraram. Evita que um Register chegue
	// depois do Unregister (corrida de desconexão rápida) e crie um player
	// fantasma que nunca seria removido da sala.
	closed atomic.Bool

	// closeOnce protege o transporte/contador; unregisterOnce protege o evento.
	closeOnce      sync.Once
	unregisterOnce sync.Once
}

// Hub gerencia as conexões de clientes e as salas de jogo (múltiplas partidas).
type Hub struct {
	Rooms        map[string]*Room
	roomSeq      int
	Register     chan *Client
	Unregister   chan *Client
	Command      chan PlayerCommand   // Canal para comandos de entrada dos jogadores
	ScoreService service.ScoreService // Serviço para lidar com o placar

	// ConnCount contabiliza conexões ativas e vagas reservadas durante o handshake.
	// Protege o servidor contra exaustão de conexões.
	ConnCount atomic.Int64
	stopping  chan struct{}
	finished  chan struct{}
	lifecycle sync.Mutex
	pumps     sync.WaitGroup
	scores    sync.WaitGroup
}

// NewHub recebe o serviço de placar para injeção de dependência
func NewHub(ss service.ScoreService) *Hub {
	return &Hub{
		Rooms:        make(map[string]*Room),
		Register:     make(chan *Client),
		Unregister:   make(chan *Client),
		Command:      make(chan PlayerCommand),
		ScoreService: ss,
		stopping:     make(chan struct{}),
		finished:     make(chan struct{}),
	}
}

// assignRoom aloca um cliente em uma sala com vagas. Auto-associação: o
// jogador entra na primeira sala com menos de PlayersPerRoom; se todas
// estiverem lotadas, uma nova sala é criada. Deve rodar no loop Run().
func (h *Hub) assignRoom(client *Client) *Room {
	for _, r := range h.Rooms {
		if len(r.Clients) < PlayersPerRoom {
			return r
		}
	}
	h.roomSeq++
	id := "sala_" + strconv.Itoa(h.roomSeq)
	r := newRoom(id, h.ScoreService)
	r.scoreTasks = &h.scores
	h.Rooms[id] = r
	return r
}

// removeClient remove o cliente da sua sala e, se a
// sala ficar vazia, descarta-a. Deve ser chamado a partir da goroutine Run().
func (h *Hub) removeClient(client *Client) {
	room := client.Room
	if room == nil {
		return
	}
	room.removeClient(client)
	if room.isEmpty() {
		delete(h.Rooms, room.ID)
	}
}

// sanitizeName limpa e limita o nome do jogador: remove caracteres de
// controle (ex.: \n) — evita injeção em logs (log poisoning) — e trunca
// para 20 runes (correto para Unicode).
func sanitizeName(name string) string {
	clean := strings.Map(func(r rune) rune {
		if r < 0x20 && r != ' ' {
			return -1
		}
		return r
	}, strings.TrimSpace(name))
	runes := []rune(clean)
	if len(runes) > 20 {
		runes = runes[:20]
	}
	return string(runes)
}

// Run é o único goroutine que lê/escreve os GameStates e as listas de clientes
// das salas. Todos os eventos (registro, comandos, desconexão e ticks de
// física) passam por este loop, evitando condições de corrida nos mapas
// compartilhados. Panics internos são capturados para não derrubar o servidor.
func (h *Hub) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Millisecond * 16) // ~60 FPS
	defer ticker.Stop()
	defer close(h.finished)
	defer func() {
		h.lifecycle.Lock()
		close(h.stopping)
		h.lifecycle.Unlock()
		for _, room := range h.Rooms {
			for client := range room.Clients {
				client.Close()
				h.removeClient(client)
			}
		}
		h.pumps.Wait()
		h.scores.Wait()
	}()

	for {
		if ctx.Err() != nil {
			return
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("PANIC capturado em Hub.Run: %v\n%s", r, debug.Stack())
				}
			}()
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				h.tick(now)

			case client := <-h.Register:
				// Conexão pode já ter encerrado (corrida: Unregister consumido
				// antes deste Register). Não cria player fantasma nesse caso.
				if client.closed.Load() {
					client.Conn.Close()
					return
				}
				room := h.assignRoom(client)
				room.addPlayer(client)
				log.Printf("Novo cliente na sala %s. Clientes: %d/5", room.ID, len(room.Clients))

			case client := <-h.Unregister:
				h.removeClient(client)
				log.Printf("Cliente desconectado. Conexões ativas: %d", h.ConnCount.Load())

			case playerCmd := <-h.Command:
				if playerCmd.Client.closed.Load() {
					return
				}
				room := playerCmd.Client.Room
				if room != nil {
					room.handleCommand(playerCmd.Client, playerCmd.Cmd)
				}
			}
		}()
	}
}

// Wait aguarda o encerramento de conexões e gravações pendentes.
func (h *Hub) Wait(ctx context.Context) error {
	select {
	case <-h.finished:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ReserveConnection reserva atomicamente uma vaga, inclusive durante o handshake.
func (h *Hub) ReserveConnection() bool {
	h.lifecycle.Lock()
	defer h.lifecycle.Unlock()
	select {
	case <-h.stopping:
		return false
	default:
	}
	for {
		n := h.ConnCount.Load()
		if n >= MaxClients {
			return false
		}
		if h.ConnCount.CompareAndSwap(n, n+1) {
			return true
		}
	}
}

// ServeClient registra uma conexão cuja vaga já foi reservada e inicia seus pumps.
func (h *Hub) ServeClient(conn *websocket.Conn) {
	c := &Client{Hub: h, Conn: conn, Send: make(chan []byte, 8)}
	h.lifecycle.Lock()
	select {
	case <-h.stopping:
		h.lifecycle.Unlock()
		h.ConnCount.Add(-1)
		conn.Close()
		return
	default:
		h.pumps.Add(2)
	}
	h.lifecycle.Unlock()
	select {
	case h.Register <- c:
	case <-h.stopping:
		h.ConnCount.Add(-1)
		conn.Close()
		h.pumps.Done()
		h.pumps.Done()
		return
	}
	go func() { defer h.pumps.Done(); ReadPump(c) }()
	go func() { defer h.pumps.Done(); WritePump(c) }()
}

// tick executa um passo de jogo de todas as salas e descarta salas vazias.
func (h *Hub) tick(now time.Time) {
	for id, room := range h.Rooms {
		if room.isEmpty() {
			delete(h.Rooms, id)
			continue
		}
		room.tick(now)
	}
}
