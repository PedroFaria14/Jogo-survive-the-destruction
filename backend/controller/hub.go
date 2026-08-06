package controller

import (
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

	// closeOnce garante que o Unregister + fechamento da conexão sejam
	// enviados exatamente uma vez, vindos de qualquer um dos pumps.
	closeOnce sync.Once
}

// Hub gerencia as conexões de clientes e as salas de jogo (múltiplas partidas).
type Hub struct {
	Rooms        map[string]*Room
	roomSeq      int
	Register     chan *Client
	Unregister   chan *Client
	Broadcast    chan []byte
	Command      chan PlayerCommand // Canal para comandos de entrada dos jogadores
	ScoreService service.ScoreService // Serviço para lidar com o placar

	// ConnCount é o número de conexões ativas (incrementado no serveWs após o
	// upgrade e decrementado exatamente uma vez no encerramento de cada client).
	// Protege o servidor contra exaustão de conexões.
	ConnCount atomic.Int64
}

// NewHub recebe o serviço de placar para injeção de dependência
func NewHub(ss service.ScoreService) *Hub {
	return &Hub{
		Rooms:        make(map[string]*Room),
		Register:     make(chan *Client),
		Unregister:   make(chan *Client),
		Broadcast:    make(chan []byte),
		Command:      make(chan PlayerCommand),
		ScoreService: ss,
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
	h.Rooms[id] = r
	return r
}

// removeClient remove o cliente da sua sala (fecha o canal Send nela) e, se a remove o cliente da sua sala (fecha o canal Send nela) e, se a
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
func (h *Hub) Run() {
	ticker := time.NewTicker(time.Millisecond * 16) // ~60 FPS
	defer ticker.Stop()

	for {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("PANIC capturado em Hub.Run: %v\n%s", r, debug.Stack())
				}
			}()
			select {
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
				room := playerCmd.Client.Room
				if room != nil {
					room.handleCommand(playerCmd.Client, playerCmd.Cmd)
				}
			}
		}()
	}
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