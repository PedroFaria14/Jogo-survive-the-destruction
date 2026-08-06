package controller

import (
	"encoding/json"
	"game-backend/models"
	"game-backend/service"
	"log"
	"time"
)

// PlayersPerRoom é o limite de jogadores simultâneos por sala/partida.
// Quando uma sala atinge esse teto, jogadores novos vão para uma nova sala.
// (A proteção global de conexões contínua sendo MaxClients no Hub.)
const PlayersPerRoom = 5

// Room representa uma partida independente: seu próprio GameState (arena,
// física e rodadas), o conjunto de clientes conectados a ela e o estado local
// de contagem regressiva da rodada. Cada sala é isolada das demais.
type Room struct {
	ID        string
	GameState *models.GameState
	Clients   map[*Client]bool

	ScoreService service.ScoreService // Serviço de placar (injetado pelo Hub)

	roundOver bool
	roundEnd  time.Time
}

// newRoom cria uma sala vazia com um estado de jogo novo.
func newRoom(id string, ss service.ScoreService) *Room {
	return &Room{
		ID:           id,
		GameState:    models.NewGameState(),
		Clients:      make(map[*Client]bool),
		ScoreService: ss,
	}
}

// addPlayer registra a conexão na sala, cria o player no GameState e associa o
// ID ao client. Deve ser chamado a partir da goroutine Run() (serializado).
func (r *Room) addPlayer(client *Client) {
	client.Room = r
	r.Clients[client] = true
	player := r.GameState.AddPlayer(client.Conn.RemoteAddr().String())
	client.PlayerID = player.ID

	initMsg, err := json.Marshal(map[string]string{
		"type":      "init",
		"player_id": player.ID,
		"room_id":   r.ID,
	})
	if err != nil {
		log.Printf("Erro ao serializar init: %v", err)
		return
	}
	client.Send <- initMsg
}

// removeClient tira o jogador da sala (fecha o canal Send apenas nesta sala).
// Deve ser chamado a partir da goroutine do Hub (serializado).
func (r *Room) removeClient(client *Client) {
	if _, ok := r.Clients[client]; ok {
		delete(r.Clients, client)
		close(client.Send)
	}
	r.GameState.RemovePlayer(client.PlayerID)
	client.Room = nil
}

// isEmpty indica que a sala não possui mais jogadores e pode ser descartada.
func (r *Room) isEmpty() bool {
	return len(r.Clients) == 0
}

// RoomBroadcast empacota o GameState com metadados da sala para o frontend.
// O embedding mantém os campos do GameState no topo do JSON (o front lê
// data.players, data.round etc. sem mudanças) e adiciona a identificação da
// sala para exibição no HUD.
type RoomBroadcast struct {
	*models.GameState
	RoomID       string `json:"room_id"`
	RoomPlayers  int    `json:"room_players"`
	RoomCapacity int    `json:"room_capacity"`
}

// broadcastState serializa e envia o estado desta sala apenas para os seus
// clientes. Cliente lento é removido da sala e tem a conexão fechada.
func (r *Room) broadcastState() {
	stateJSON, err := json.Marshal(&RoomBroadcast{
		GameState:    r.GameState,
		RoomID:       r.ID,
		RoomPlayers:  len(r.Clients),
		RoomCapacity: PlayersPerRoom,
	})
	if err != nil {
		log.Printf("Erro ao serializar GameState da sala %s: %v", r.ID, err)
		return
	}
	for client := range r.Clients {
		select {
		case client.Send <- stateJSON:
		default:
			r.removeClient(client)
			client.Conn.Close()
		}
	}
}

// saveFinalScores salva (uma única vez) o placar dos jogadores ao final de
// cada participação: os mortos durante a rodada ativa e, quando a rodada
// encerra (roundOver), também os que permaneceram vivos (vencedores).
// Não fecha a conexão — o jogador vira espectador.
func (r *Room) saveFinalScores() {
	for _, p := range r.GameState.Players {
		if p.ScoreSaved {
			continue
		}
		// Durante a rodada ativa, apenas mortos têm placar final. Ao encerrar
		// a rodada, os vivos também contam (venceram/sobreviveram ao colapso).
		if !p.IsDead && !r.roundOver {
			continue
		}
		p.ScoreSaved = true
		duration := time.Since(p.StartTime)
		finalScore := int(duration.Seconds())
		// Filtro anti-spam: placares triviais (restart em loop) não poluem
		// o Top 10 nem crescem o banco sem limite.
		if finalScore < MinScoreToSave {
			continue
		}
		name := p.Name
		if name == "" {
			name = p.ID
		}

		pp := p
		go func() {
			if err := r.ScoreService.SaveScore(pp.ID, name, finalScore, duration); err != nil {
				log.Printf("ERRO ao salvar placar do jogador %s: %v", pp.ID, err)
			}
		}()
	}
}

// roundShouldEnd indica o fim da rodada: arena sem blocos ativos, todos os
// jogadores eliminados, ou (em multiplayer) resta apenas um jogador vivo —
// o último sobrevivente vence a rodada.
func (r *Room) roundShouldEnd() bool {
	gs := r.GameState
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

// tick executa um passo de jogo da sala. Roda dentro da goroutine Run() do
// Hub (serializado entre salas).
func (r *Room) tick(now time.Time) {
	gs := r.GameState

	// 1. Detecta o fim de rodada: todos mortos ou arena sem blocos ativos.
	if !r.roundOver && r.roundShouldEnd() {
		r.roundOver = true
		r.roundEnd = now.Add(RoundRestartDelay)
		gs.RoundOver = true
		gs.Status = "round_over"
		log.Printf("Salas %s: rodada encerrada. Nova rodada em %s.", r.ID, RoundRestartDelay)
	}

	// 2. Contagem regressiva / reinício.
	if r.roundOver {
		r.saveFinalScores()

		remaining := int(r.roundEnd.Sub(now).Seconds() + 0.999)
		if remaining < 0 {
			remaining = 0
		}
		gs.Countdown = remaining

		if !now.Before(r.roundEnd) {
			gs.ResetRound()
			r.roundOver = false
			r.roundEnd = time.Time{}
		}
		r.broadcastState()
		return
	}

	// 3. Rodada ativa: física, power-ups, destruição e placares.
	r.saveFinalScores()
	gs.UpdatePowerUps(now)
	gs.ApplyPhysics()
	gs.CheckArenaDestruction()
	gs.ExpireFallingTiles(now)
	gs.SpawnLostTile(now)
	for _, p := range gs.Players {
		if !p.IsDead {
			p.Score = int(now.Sub(p.StartTime).Seconds())
		}
	}

	r.broadcastState()
}

// handleCommand processa um comando de um jogador desta sala.
func (r *Room) handleCommand(client *Client, cmd models.Command) {
	gs := r.GameState

	// Comando de join: define o nickname (sanitizado) e a cor da bolinha.
	if cmd.Type == "join" {
		name := sanitizeName(cmd.Name)
		color := models.SanitizeColor(cmd.Color)
		if p := gs.Players[client.PlayerID]; p != nil {
			p.Name = name
			p.Color = color
		}
		return
	}
	// Comando de restart: respawna o jogador no mesmo mapa com vidas cheias.
	// Apenas jogadores mortos (evita exploit de cura/teleporte).
	if cmd.Type == "restart" {
		if !r.roundOver {
			if p := gs.Players[client.PlayerID]; p != nil && p.IsDead {
				p.Lives = models.MaxLives
				gs.RespawnPlayer(client.PlayerID)
			}
		}
		return
	}
	// Demais comandos: input de movimento.
	gs.ProcessInput(client.PlayerID, cmd)
}