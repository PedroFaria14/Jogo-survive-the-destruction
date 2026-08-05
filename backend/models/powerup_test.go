package models

import (
	"testing"
	"time"
)

// TestUpdatePowerUpsPickup garante que um jogador sem buff coleta o drop e o
// drop é removido da arena.
func TestUpdatePowerUpsPickup(t *testing.T) {
	gs := NewGameState()
	p := gs.AddPlayer("conn1")
	now := time.Now()
	gs.PowerUps["pu1"] = &PowerUp{
		ID:      "pu1",
		Type:    PowerUpRedMushroom,
		X:       p.X,
		Y:       p.Y,
		SpawnAt: now,
	}

	gs.UpdatePowerUps(now)

	if p.Buff != PowerUpRedMushroom {
		t.Fatalf("Buff = %q, esperado %q", p.Buff, PowerUpRedMushroom)
	}
	if _, exists := gs.PowerUps["pu1"]; exists {
		t.Fatal("drop deveria ter sido removido após o pickup")
	}
}

// TestUpdatePowerUpsBuffExpiry garante que um buff vencido é removido.
func TestUpdatePowerUpsBuffExpiry(t *testing.T) {
	gs := NewGameState()
	p := gs.AddPlayer("conn1")
	p.Buff = PowerUpBlueCrystal
	p.BuffUntil = time.Now().Add(-time.Second)

	gs.UpdatePowerUps(time.Now())

	if p.Buff != "" {
		t.Fatalf("Buff = %q, esperado vazio após expirar", p.Buff)
	}
	if p.BuffRemaining != 0 {
		t.Fatalf("BuffRemaining = %v, esperado 0", p.BuffRemaining)
	}
}

// TestUpdatePowerUpsDespawn garante que drops que venceram o tempo de vida
// são removidos da arena.
func TestUpdatePowerUpsDespawn(t *testing.T) {
	gs := NewGameState()
	now := time.Now()
	gs.PowerUps["pu1"] = &PowerUp{
		ID:      "pu1",
		Type:    PowerUpRedMushroom,
		X:       100,
		Y:       100,
		SpawnAt: now.Add(-2 * DropLifetime),
	}

	gs.UpdatePowerUps(now)

	if len(gs.PowerUps) != 0 {
		t.Fatalf("drops após despawn = %d, esperado 0", len(gs.PowerUps))
	}
}

// TestUpdatePowerUpsNoPickupWhenBuffed garante que quem já está buffado não
// coleta outro drop (um buff por vez) — o drop permanece na arena.
func TestUpdatePowerUpsNoPickupWhenBuffed(t *testing.T) {
	gs := NewGameState()
	p := gs.AddPlayer("conn1")
	now := time.Now()
	p.Buff = PowerUpPurpleMushroom
	p.BuffUntil = now.Add(time.Minute)
	gs.PowerUps["pu1"] = &PowerUp{
		ID:      "pu1",
		Type:    PowerUpRedMushroom,
		X:       p.X,
		Y:       p.Y,
		SpawnAt: now,
	}

	gs.UpdatePowerUps(now)

	if _, exists := gs.PowerUps["pu1"]; !exists {
		t.Fatal("drop deveria permanecer: jogador buffado não coleta outro")
	}
}
