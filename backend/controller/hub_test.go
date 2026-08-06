package controller

import (
	"game-backend/service"
	"sync"
	"testing"
	"time"
)

// mockScoreService registra chamadas de SaveScore em memória (sem banco).
type mockScoreService struct {
	mu    sync.Mutex
	saves []service.Score
}

func (m *mockScoreService) GetTopScores() ([]service.Score, error) { return nil, nil }
func (m *mockScoreService) SaveScore(playerID, playerName string, scoreSeconds int, duration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.saves = append(m.saves, service.Score{PlayerID: playerID, Name: playerName, ScoreSec: scoreSeconds})
	return nil
}
func (m *mockScoreService) Close() error { return nil }

// count retorna o número de placares salvos até o momento.
func (m *mockScoreService) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.saves)
}

// waitSaves aguarda até que ao menos n placares sejam salvos (as goroutines
// de SaveScore são assíncronas) ou estoura o timeout.
func (m *mockScoreService) waitSaves(n int) {
	deadline := time.Now().Add(2 * time.Second)
	for m.count() < n {
		if time.Now().After(deadline) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// newTestHub cria um Hub sem serviço de placar.
func newTestHub() *Hub {
	return NewHub(nil)
}

func newTestHubWithScores(m *mockScoreService) *Hub {
	return NewHub(m)
}

// newTestRoom cria uma sala de teste (com serviço de placar opcional).
func newTestRoom(ss service.ScoreService) *Room {
	return newRoom("sala_teste", ss)
}

func TestSanitizeName(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "nome normal", in: "  Pedro  ", want: "Pedro"},
		{name: "remove quebra de linha", in: "Pedro\nAdmin", want: "PedroAdmin"},
		{name: "remove tab", in: "A\tB", want: "AB"},
		{name: "mantém espaços internos", in: "Pedro Lucas", want: "Pedro Lucas"},
		{name: "trunca 20 runes", in: "abcdefghijklmnopqrstuvwxyz", want: "abcdefghijklmnopqrst"},
		{name: "vazio", in: "   ", want: ""},
	}
	for _, c := range cases {
		if got := sanitizeName(c.in); got != c.want {
			t.Errorf("sanitizeName(%q) = %q, esperado %q", c.in, got, c.want)
		}
	}
}

// TestSaveFinalScoresWinner garante que, com a rodada encerrada, o placar do
// vencedor (jogador ainda vivo) é persistido.
func TestSaveFinalScoresWinner(t *testing.T) {
	mock := &mockScoreService{}
	room := newTestRoom(mock)
	p := room.GameState.AddPlayer("conn1")
	p.Name = "Campea"
	p.StartTime = time.Now().Add(-10 * time.Second)
	room.roundOver = true

	room.saveFinalScores()
	mock.waitSaves(1)

	if mock.count() != 1 {
		t.Fatalf("placares salvos = %d, esperado 1 (vencedor)", mock.count())
	}
}

// TestSaveFinalScoresDeadDuringRound garante que, durante a rodada ativa, o
// placar de um jogador morto é salvo e o de um vivo não.
func TestSaveFinalScoresDeadDuringRound(t *testing.T) {
	mock := &mockScoreService{}
	room := newTestRoom(mock)
	dead := room.GameState.AddPlayer("conn1")
	alive := room.GameState.AddPlayer("conn2")
	dead.IsDead = true
	dead.StartTime = time.Now().Add(-10 * time.Second)
	alive.IsDead = false
	alive.StartTime = time.Now().Add(-10 * time.Second)
	room.roundOver = false

	room.saveFinalScores()
	mock.waitSaves(1)

	if mock.count() != 1 {
		t.Fatalf("placares salvos = %d, esperado 1 (só o morto)", mock.count())
	}
	if alive.ScoreSaved {
		t.Fatal("jogador vivo não deveria ter placar salvo durante a rodada")
	}
}

// TestSaveFinalScoresSkipsTrivial garante que placares abaixo do threshold
// (spam de restart) não são persistidos.
func TestSaveFinalScoresSkipsTrivial(t *testing.T) {
	mock := &mockScoreService{}
	room := newTestRoom(mock)
	p := room.GameState.AddPlayer("conn1")
	p.IsDead = true
	p.StartTime = time.Now() // duration < MinScoreToSave

	room.saveFinalScores()
	time.Sleep(50 * time.Millisecond)

	if mock.count() != 0 {
		t.Fatalf("placares salvos = %d, esperado 0 (placar trivial)", mock.count())
	}
}

func TestRoundShouldEndNoPlayers(t *testing.T) {
	room := newTestRoom(nil)
	if room.roundShouldEnd() {
		t.Fatal("rodada não deve encerrar sem jogadores")
	}
}

func TestRoundShouldEndAllDead(t *testing.T) {
	room := newTestRoom(nil)
	room.GameState.AddPlayer("conn1")
	for _, p := range room.GameState.Players {
		p.IsDead = true
	}
	if !room.roundShouldEnd() {
		t.Fatal("rodada deve encerrar com todos os jogadores mortos")
	}
}

func TestRoundShouldEndSoloAlive(t *testing.T) {
	room := newTestRoom(nil)
	p := room.GameState.AddPlayer("conn1")
	p.IsDead = false
	// Total == 1: não pode encerrar só porque resta 1 vivo.
	if room.roundShouldEnd() {
		t.Fatal("rodada não deve encerrar com 1 jogador vivo em jogo solo")
	}
}

func TestRoundShouldEndMultiLastAlive(t *testing.T) {
	room := newTestRoom(nil)
	a := room.GameState.AddPlayer("conn1")
	b := room.GameState.AddPlayer("conn2")
	a.IsDead = false
	b.IsDead = true
	if !room.roundShouldEnd() {
		t.Fatal("rodada deve encerrar com apenas 1 vivo em multiplayer")
	}
}

func TestRoundShouldEndMultiAllAlive(t *testing.T) {
	room := newTestRoom(nil)
	a := room.GameState.AddPlayer("conn1")
	b := room.GameState.AddPlayer("conn2")
	a.IsDead = false
	b.IsDead = false
	if room.roundShouldEnd() {
		t.Fatal("rodada não deve encerrar com todos vivos")
	}
}

func TestRoundShouldEndNoActiveTiles(t *testing.T) {
	room := newTestRoom(nil)
	room.GameState.AddPlayer("conn1")
	// Desativa todos os tiles: mesmo com jogador vivo, a arena acabou.
	for _, tile := range room.GameState.ArenaTiles {
		tile.IsActive = false
	}
	if !room.roundShouldEnd() {
		t.Fatal("rodada deve encerrar quando a arena fica sem tiles ativos")
	}
}

// TestAssignRoomReusesOpen garante que um novo cliente entra na primeira sala
// que ainda tem vagas, em vez de criar uma nova.
func TestAssignRoomReusesOpen(t *testing.T) {
	h := newTestHub()
	roomA := h.assignRoom(&Client{})
	if roomA == nil {
		t.Fatal("assignRoom não deveria retornar nil")
	}
	// Simula 4 jogadores na sala A (5 - 1 = 4 vagas restantes).
	clients := make([]*Client, PlayersPerRoom-1)
	for i := range clients {
		clients[i] = &Client{}
		roomA.Clients[clients[i]] = true
	}
	// O 5º jogador ainda cabe na sala A.
	fifth := h.assignRoom(&Client{})
	if fifth != roomA {
		t.Fatalf("5º jogador deveria entrar na sala existente %q, mas entrou em outra", roomA.ID)
	}
}

// TestAssignRoomCreatesNewWhenFull garante que, com a sala lotada, o jogador
// entra em uma sala nova.
func TestAssignRoomCreatesNewWhenFull(t *testing.T) {
	h := newTestHub()
	roomA := h.assignRoom(&Client{})
	clients := make([]*Client, PlayersPerRoom)
	for i := range clients {
		clients[i] = &Client{}
		roomA.Clients[clients[i]] = true
	}
	if len(roomA.Clients) != PlayersPerRoom {
		t.Fatalf("sala deveria ter %d clientes, tem %d", PlayersPerRoom, len(roomA.Clients))
	}
	next := h.assignRoom(&Client{})
	if next == roomA {
		t.Fatal("jogador não deveria entrar em sala lotada")
	}
	if len(h.Rooms) != 2 {
		t.Fatalf("deveria haver 2 salas, há %d", len(h.Rooms))
	}
}

// TestEmptyRoomCleanupRemove garante que remover o último cliente descarta
// a sala do Hub.
func TestEmptyRoomCleanupRemove(t *testing.T) {
	h := newTestHub()
	client := &Client{Send: make(chan []byte)}
	room := h.assignRoom(client)
	client.Room = room // addPlayer faria isso na conexão real
	room.Clients[client] = true

	h.removeClient(client)
	if _, ok := h.Rooms[room.ID]; ok {
		t.Fatal("sala vazia deveria ser removida do Hub")
	}
}

// TestPlayerIDsMayCollideAcrossRooms documenta que o PlayerID não é único
// globalmente (cada sala tem seu próprio contador). Por isso o roteamento de
// comandos usa Client.Room, e nunca o PlayerID sozinho.
func TestPlayerIDsMayCollideAcrossRooms(t *testing.T) {
	roomA := newTestRoom(nil)
	roomB := newTestRoom(nil)
	pa := roomA.GameState.AddPlayer("conn1")
	pb := roomB.GameState.AddPlayer("conn1")
	if pa.ID != pb.ID {
		t.Fatalf("IDs por sala deveriam colidir (ambos %q)", pa.ID)
	}
}