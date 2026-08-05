package models

import (
	"fmt"
	"log"
	"math"
	"time"
)

// Tipos de drop/power-up.
const (
	PowerUpRedMushroom    = "red_mushroom"    // Tanque: dobra tamanho e massa
	PowerUpPurpleMushroom = "purple_mushroom" // Velocista: multiplica a velocidade horizontal
	PowerUpBlueCrystal    = "blue_crystal"    // Planar: gravidade lunar enquanto no ar
)

// Parâmetros dos buffs (segundos).
const (
	BuffRedDuration    = 8 * time.Second
	BuffPurpleDuration = 6 * time.Second
	BuffBlueDuration   = 8 * time.Second
)

// Cadência e física dos drops.
const (
	DropInterval  = 4 * time.Second // Um drop a cada intervalo
	DropLifetime  = 6 * time.Second // Some se ninguém pegar
	DropRadius    = 16.0
	DropGravity   = 1.0
	DropMaxSpeedY = 10.0 // Evita tunneling durante a queda
)

// Efeitos numéricos dos buffs.
const (
	TankRadiusScale = 2.0 // Raio do Tanque (dobra o tamanho)
	TankMass        = 2.0 // Massa do Tanque
	SpeedMult       = 1.8 // Multiplicador de velocidade do Velocista
	GlideGravity    = 0.2 // Escala de gravidade do Planar
)

// powerUpTypes são os tipos sorteados a cada spawn.
var powerUpTypes = []string{PowerUpRedMushroom, PowerUpPurpleMushroom, PowerUpBlueCrystal}

// PowerUp é um item coletável que cai do céu sobre uma ilha.
type PowerUp struct {
	ID      string    `json:"id"`
	Type    string    `json:"type"`
	X       float64   `json:"x"`
	Y       float64   `json:"y"`
	Vy      float64   `json:"-"`
	Landed  bool      `json:"landed"`
	SpawnAt time.Time `json:"-"`
}

// buffActive indica se o jogador está com o buff vigente agora.
func (p *Player) buffActive(now time.Time) bool {
	return p.Buff != "" && now.Before(p.BuffUntil)
}

// applyBuff aplica o power-up, reiniciando o temporizador do buff.
func (p *Player) applyBuff(typ string, now time.Time) {
	p.Buff = typ
	switch typ {
	case PowerUpRedMushroom:
		p.BuffUntil = now.Add(BuffRedDuration)
	case PowerUpPurpleMushroom:
		p.BuffUntil = now.Add(BuffPurpleDuration)
	case PowerUpBlueCrystal:
		p.BuffUntil = now.Add(BuffBlueDuration)
	}
	p.BuffRemaining = p.BuffUntil.Sub(now).Seconds()
}

// radius retorna o raio atual do jogador (o Tanque dobra de tamanho).
func (p *Player) radius(now time.Time) float64 {
	if p.Buff == PowerUpRedMushroom && p.buffActive(now) {
		return PlayerRadius * TankRadiusScale
	}
	return PlayerRadius
}

// mass retorna a massa atual do jogador (o Tanque fica pesado).
func (p *Player) mass() float64 {
	if p.Buff == PowerUpRedMushroom && p.buffActive(time.Now()) {
		return TankMass
	}
	return 1.0
}

// speedMult retorna o multiplicador de velocidade horizontal do jogador.
func (p *Player) speedMult() float64 {
	if p.Buff == PowerUpPurpleMushroom && p.buffActive(time.Now()) {
		return SpeedMult
	}
	return 1.0
}

// gravityScale retorna a escala de gravidade do jogador (o Planar flutua).
func (p *Player) gravityScale() float64 {
	if p.Buff == PowerUpBlueCrystal && p.buffActive(time.Now()) {
		return GlideGravity
	}
	return 1.0
}

// UpdatePowerUps avança o ciclo de drops e buffs. Deve ser chamado pela
// goroutine do Hub (serializado), antes da física da rodada.
func (gs *GameState) UpdatePowerUps(now time.Time) {
	// 1. Expira buffs vencidos (antes da física usar os multiplicadores).
	for _, p := range gs.Players {
		if p.Buff != "" && !p.buffActive(now) {
			log.Printf("Buff %q do jogador %s expirou.", p.Buff, p.ID)
			p.Buff = ""
			p.BuffUntil = time.Time{}
		}
	}

	// 2. Spawna um novo drop quando o intervalo tiver passado.
	gs.DropCountdown = math.Max(0, time.Until(gs.nextDropAt).Seconds())
	if !now.Before(gs.nextDropAt) {
		gs.spawnPowerUp(now)
		gs.nextDropAt = now.Add(DropInterval)
		gs.DropCountdown = DropInterval.Seconds()
	}

	// 3. Física da queda dos drops até pousarem numa ilha.
	for _, pu := range gs.PowerUps {
		if pu.Landed {
			continue
		}
		pu.Vy = math.Min(pu.Vy+DropGravity, DropMaxSpeedY)
		pu.Y += pu.Vy
		for _, t := range gs.ArenaTiles {
			if !t.IsActive {
				continue
			}
			if pu.X > t.X && pu.X < t.X+TileSize &&
				pu.Y+DropRadius >= t.Y && pu.Y+DropRadius < t.Y+TileSize {
				pu.Y = t.Y - DropRadius
				pu.Vy = 0
				pu.Landed = true
				break
			}
		}
	}

	// 4. Despawn: venceu o tempo de vida ou caiu além da killzone.
	for id, pu := range gs.PowerUps {
		if now.Sub(pu.SpawnAt) > DropLifetime || pu.Y > gs.ArenaHeight+100 {
			delete(gs.PowerUps, id)
		}
	}

	// 5. Pickup: jogador sem buff encosta no drop e o coleta. Quem já está
	// buffado NÃO consegue pegar outro — o drop permanece para os demais.
	for _, p := range gs.Players {
		if p.IsDead {
			continue
		}
		for id, pu := range gs.PowerUps {
			dx := pu.X - p.X
			dy := pu.Y - p.Y
			if math.Hypot(dx, dy) < p.radius(now)+DropRadius {
				if p.Buff == "" {
					p.applyBuff(pu.Type, now)
					delete(gs.PowerUps, id)
					log.Printf("Jogador %s pegou %s.", p.ID, pu.Type)
				}
				break
			}
		}
	}

	// 6. Atualiza os valores de HUD.
	for _, p := range gs.Players {
		if p.Buff != "" && p.buffActive(now) {
			p.BuffRemaining = math.Max(0, time.Until(p.BuffUntil).Seconds())
		} else {
			p.BuffRemaining = 0
		}
	}
}

// spawnPowerUp sorteia um tipo e o solta do céu sobre um tile de superfície.
func (gs *GameState) spawnPowerUp(now time.Time) {
	topTiles := make([]*ArenaTile, 0, 8)
	for _, t := range gs.ArenaTiles {
		if t.IsActive && !t.IsFalling && t.Kind == "top" {
			topTiles = append(topTiles, t)
		}
	}
	if len(topTiles) == 0 {
		return
	}

	tile := topTiles[rng.Intn(len(topTiles))]
	gs.nextPowerUpID++
	id := fmt.Sprintf("powerup_%d", gs.nextPowerUpID)
	gs.PowerUps[id] = &PowerUp{
		ID:      id,
		Type:    powerUpTypes[rng.Intn(len(powerUpTypes))],
		X:       tile.X + TileSize/2,
		Y:       tile.Y - DropRadius - 60,
		SpawnAt: now,
	}
	log.Printf("Drop %s (%s) apareceu em (%.1f, %.1f).", id, gs.PowerUps[id].Type, gs.PowerUps[id].X, gs.PowerUps[id].Y)
}

// clearPowerUps remove todos os drops da arena (usado no reset de rodada).
func (gs *GameState) clearPowerUps() {
	gs.PowerUps = make(map[string]*PowerUp)
	gs.nextPowerUpID = 0
	gs.nextDropAt = time.Now().Add(DropInterval)
	gs.DropCountdown = DropInterval.Seconds()
}
