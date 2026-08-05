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

// newTestHub cria um Hub sem serviço de placar (não usado por roundShouldEnd).
func newTestHub() *Hub {
	return NewHub(nil)
}

func newTestHubWithScores(m *mockScoreService) *Hub {
	return NewHub(m)
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
	h := newTestHubWithScores(mock)
	p := h.GameState.AddPlayer("conn1")
	p.Name = "Campea"
	p.StartTime = time.Now().Add(-10 * time.Second)
	h.roundOver = true

	h.saveFinalScores()
	mock.waitSaves(1)

	if mock.count() != 1 {
		t.Fatalf("placares salvos = %d, esperado 1 (vencedor)", mock.count())
	}
}

// TestSaveFinalScoresDeadDuringRound garante que, durante a rodada ativa, o
// placar de um jogador morto é salvo e o de um vivo não.
func TestSaveFinalScoresDeadDuringRound(t *testing.T) {
	mock := &mockScoreService{}
	h := newTestHubWithScores(mock)
	dead := h.GameState.AddPlayer("conn1")
	alive := h.GameState.AddPlayer("conn2")
	dead.IsDead = true
	dead.StartTime = time.Now().Add(-10 * time.Second)
	alive.IsDead = false
	alive.StartTime = time.Now().Add(-10 * time.Second)
	h.roundOver = false

	h.saveFinalScores()
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
	h := newTestHubWithScores(mock)
	p := h.GameState.AddPlayer("conn1")
	p.IsDead = true
	p.StartTime = time.Now() // duration < MinScoreToSave

	h.saveFinalScores()
	time.Sleep(50 * time.Millisecond)

	if mock.count() != 0 {
		t.Fatalf("placares salvos = %d, esperado 0 (placar trivial)", mock.count())
	}
}

func TestRoundShouldEndNoPlayers(t *testing.T) {
	h := newTestHub()
	if h.roundShouldEnd() {
		t.Fatal("rodada não deve encerrar sem jogadores")
	}
}

func TestRoundShouldEndAllDead(t *testing.T) {
	h := newTestHub()
	h.GameState.AddPlayer("conn1")
	for _, p := range h.GameState.Players {
		p.IsDead = true
	}
	if !h.roundShouldEnd() {
		t.Fatal("rodada deve encerrar com todos os jogadores mortos")
	}
}

func TestRoundShouldEndSoloAlive(t *testing.T) {
	h := newTestHub()
	p := h.GameState.AddPlayer("conn1")
	p.IsDead = false
	// Total == 1: não pode encerrar só porque resta 1 vivo.
	if h.roundShouldEnd() {
		t.Fatal("rodada não deve encerrar com 1 jogador vivo em jogo solo")
	}
}

func TestRoundShouldEndMultiLastAlive(t *testing.T) {
	h := newTestHub()
	a := h.GameState.AddPlayer("conn1")
	b := h.GameState.AddPlayer("conn2")
	a.IsDead = false
	b.IsDead = true
	if !h.roundShouldEnd() {
		t.Fatal("rodada deve encerrar com apenas 1 vivo em multiplayer")
	}
}

func TestRoundShouldEndMultiAllAlive(t *testing.T) {
	h := newTestHub()
	a := h.GameState.AddPlayer("conn1")
	b := h.GameState.AddPlayer("conn2")
	a.IsDead = false
	b.IsDead = false
	if h.roundShouldEnd() {
		t.Fatal("rodada não deve encerrar com todos vivos")
	}
}

func TestRoundShouldEndNoActiveTiles(t *testing.T) {
	h := newTestHub()
	h.GameState.AddPlayer("conn1")
	// Desativa todos os tiles: mesmo com jogador vivo, a arena acabou.
	for _, tile := range h.GameState.ArenaTiles {
		tile.IsActive = false
	}
	if !h.roundShouldEnd() {
		t.Fatal("rodada deve encerrar quando a arena fica sem tiles ativos")
	}
}
