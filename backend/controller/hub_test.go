package controller

import (
	"testing"
)

// newTestHub cria um Hub sem serviço de placar (não usado por roundShouldEnd).
func newTestHub() *Hub {
	return NewHub(nil)
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
