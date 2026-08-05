package models

import (
	"math"
	"testing"
	"time"
)

// makeIsland constrói um islandPlan com perfil em arco para testes.
func makeIsland(col, width, bottom int) islandPlan {
	return islandPlan{Col: col, Width: width, Profile: islandProfile(width), Bottom: bottom}
}

func TestSanitizeColor(t *testing.T) {
	cases := []struct {
		name  string
		in    string
		want  string
	}{
		{name: "hex minúsculo válido", in: "#ff6b5e", want: "#ff6b5e"},
		{name: "hex maiúsculo vira minúsculo", in: "#FF6B5E", want: "#ff6b5e"},
		{name: "hex com espaços é aceito", in: "  #f2b544 ", want: "#f2b544"},
		{name: "sem #", in: "f2b544", want: ""},
		{name: "comprimento inválido", in: "#f2b54", want: ""},
		{name: "caracteres não hex", in: "#gggggg", want: ""},
		{name: "vazio", in: "", want: ""},
	}
	for _, c := range cases {
		if got := SanitizeColor(c.in); got != c.want {
			t.Errorf("SanitizeColor(%q) = %q, esperado %q", c.in, got, c.want)
		}
	}
}

func TestSpawnLostTile(t *testing.T) {
	gs := NewGameState()
	gs.ArenaTiles = make(map[string]*ArenaTile)
	gs.nextLostTileAt = time.Now().Add(-time.Second)
	gs.ArenaWidth = 8 * TileSize
	gs.ArenaHeight = ArenaHeight

	before := gs.nextLostTileAt
	gs.SpawnLostTile(time.Now())

	if len(gs.ArenaTiles) != 1 {
		t.Fatalf("esperava 1 quadrado perdido, obteve %d", len(gs.ArenaTiles))
	}
	for _, tile := range gs.ArenaTiles {
		if tile.Kind != "lost" || !tile.IsActive {
			t.Fatalf("quadrado perdido inválido: %+v", tile)
		}
	}
	if !gs.nextLostTileAt.After(before) {
		t.Fatalf("timer do quadrado perdido não avançou")
	}
}

func TestSpawnLostTileCooldown(t *testing.T) {
	gs := NewGameState()
	gs.ArenaTiles = make(map[string]*ArenaTile)
	gs.nextLostTileAt = time.Now().Add(time.Hour)
	gs.SpawnLostTile(time.Now())
	if len(gs.ArenaTiles) != 0 {
		t.Fatalf("não deveria spawnar antes do timer")
	}
}

func TestIslandProfile(t *testing.T) {
	got := islandProfile(5)
	want := []int{1, 2, 3, 2, 1}
	if len(got) != len(want) {
		t.Fatalf("islandProfile(5) = %v, esperado %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("islandProfile(5)[%d] = %d, esperado %d", i, got[i], want[i])
		}
	}
	if len(islandProfile(1)) != 1 || islandProfile(1)[0] != 1 {
		t.Fatalf("islandProfile(1) = %v, esperado [1]", islandProfile(1))
	}
}

func TestCanReach(t *testing.T) {
	cases := []struct {
		name  string
		a, b  islandPlan
		reach bool
	}{
		{
			name:  "mesmo nível, gap 1 (alcançável)",
			a:     makeIsland(0, 1, 4),
			b:     makeIsland(2, 1, 4),
			reach: true,
		},
		{
			name:  "mesmo nível, gap máximo permitido",
			a:     makeIsland(0, 1, 4),
			b:     makeIsland(5, 1, 4), // gap = 5-1 = 4 = MaxGapCols+1
			reach: true,
		},
		{
			name:  "mesmo nível, gap acima do limite",
			a:     makeIsland(0, 1, 4),
			b:     makeIsland(6, 1, 4), // gap = 5 > 4
			reach: false,
		},
		{
			name:  "subida 1 tile, gap 1",
			a:     makeIsland(0, 1, 4), // minTop 4
			b:     makeIsland(1, 2, 3), // profile [1,1], minTop 3
			reach: true,
		},
		{
			name:  "subida 2 tiles, gap 1",
			a:     makeIsland(0, 1, 4), // minTop 4
			b:     makeIsland(1, 1, 2), // minTop 2, rise 2, gap 0
			reach: true,
		},
		{
			name:  "subida 2 tiles, gap 2 (inalcançável)",
			a:     makeIsland(0, 1, 4),
			b:     makeIsland(3, 1, 2), // gap = 3-1 = 2 > 1
			reach: false,
		},
		{
			name:  "subida 3 tiles, sempre inalcançável",
			a:     makeIsland(0, 1, 4), // minTop 4
			b:     makeIsland(1, 1, 1), // minTop 1, rise 3
			reach: false,
		},
		{
			name:  "descida não afeta o limite horizontal",
			a:     makeIsland(0, 1, 1), // mais baixo
			b:     makeIsland(5, 1, 4), // mais alto: rise = 1-4 = -3 (subindo em a)
			reach: true,               // canReach(a,b) com rise<=0 permite gap 4
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := canReach(tc.a, tc.b); got != tc.reach {
				t.Fatalf("canReach(a,b) = %v, esperado %v", got, tc.reach)
			}
		})
	}
}

// TestBuildLayoutReachable garante que, sempre que o gerador produz um layout
// válido, todas as ilhas são alcançáveis a partir da ilha de spawn (a "regra
// de ouro"). Falhas individuais são aceitas — o generateArena tem retry e
// fallback — mas a taxa de sucesso deve se manter alta.
func TestBuildLayoutReachable(t *testing.T) {
	const attempts = 200
	successes := 0
	for i := 0; i < attempts; i++ {
		res, ok := buildLayout()
		if !ok {
			continue // o generateArena tenta de novo / usa fallback
		}
		successes++
		if len(res.islands) < 2 {
			t.Fatalf("iteração %d: apenas %d ilhas", i, len(res.islands))
		}

		visited := make([]bool, len(res.islands))
		visited[0] = true
		queue := []int{0}
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			for j := range res.islands {
				if visited[j] {
					continue
				}
				if canReach(res.islands[cur], res.islands[j]) || canReach(res.islands[j], res.islands[cur]) {
					visited[j] = true
					queue = append(queue, j)
				}
			}
		}
		for idx, v := range visited {
			if !v {
				t.Fatalf("iteração %d: ilha %d inalcançável a partir do spawn", i, idx)
			}
		}
	}
	if successes < attempts/2 {
		t.Fatalf("taxa de sucesso do gerador muito baixa: %d/%d", successes, attempts)
	}
}

// TestGenerateArenaBounds garante que a arena gerada (ou fallback) mantém
// todos os tiles e pontos de spawn dentro dos limites do jogo.
func TestGenerateArenaBounds(t *testing.T) {
	gs := NewGameState()
	if len(gs.ArenaTiles) == 0 {
		t.Fatal("arena sem tiles")
	}
	if len(gs.SpawnPoints) == 0 {
		t.Fatal("arena sem pontos de spawn")
	}
	if gs.ArenaWidth <= 0 || gs.ArenaWidth > float64(MaxArenaCols+1)*TileSize {
		t.Fatalf("ArenaWidth inválida: %v", gs.ArenaWidth)
	}
	for id, tile := range gs.ArenaTiles {
		if tile.X < 0 || tile.X+TileSize > gs.ArenaWidth {
			t.Fatalf("tile %s fora dos limites horizontais (x=%v)", id, tile.X)
		}
		if tile.Y < 0 || tile.Y+TileSize > ArenaHeight {
			t.Fatalf("tile %s fora dos limites verticais (y=%v)", id, tile.Y)
		}
		if !tile.IsActive {
			t.Fatalf("tile %s nasceu inativo", id)
		}
	}
}

// TestApplyPhysicsFallingAndLanding verifica que um jogador spawnado cai e
// pousa em uma ilha, ficando no chão sem perder vidas.
func TestApplyPhysicsFallingAndLanding(t *testing.T) {
	gs := NewGameState()
	p := gs.AddPlayer("conn1")

	for i := 0; i < 600; i++ {
		gs.ApplyPhysics()
		if p.IsOnGround {
			break
		}
	}
	if !p.IsOnGround {
		t.Fatalf("jogador não pousou após 600 ticks (y=%.2f, vidas=%d)", p.Y, p.Lives)
	}
	if p.Lives != MaxLives {
		t.Fatalf("jogador perdeu vida sem cair na killzone: vidas=%d", p.Lives)
	}
	// Teto de velocidade deve limitar a queda (não há tunneling).
	if p.VelocityY > MaxSpeed {
		t.Fatalf("VelocityY acima do teto: %.2f", p.VelocityY)
	}
}

// TestKillzoneLosesLife verifica que cair além da killzone custa uma vida e
// respawna o jogador na arena.
func TestKillzoneLosesLife(t *testing.T) {
	gs := NewGameState()
	p := gs.AddPlayer("conn1")
	p.Lives = 2
	p.X = gs.ArenaWidth / 2
	p.Y = gs.ArenaHeight + 150 // além da killzone (ArenaHeight+100)
	p.VelocityX = 0
	p.VelocityY = 0
	p.IsOnGround = false

	gs.ApplyPhysics()

	if p.Lives != 1 {
		t.Fatalf("vidas após cair na killzone = %d, esperado 1", p.Lives)
	}
	if p.Y >= gs.ArenaHeight {
		t.Fatalf("jogador não foi respawnado na arena (y=%.2f)", p.Y)
	}
	if p.Score != 0 {
		t.Fatalf("queda deve zerar o placar, mas Score=%d", p.Score)
	}
}

// TestRespawnPlayerKeepScore garante que o teleporte anti-trava preserva o
// placar (não é uma morte), enquanto o RespawnPlayer normal zera.
func TestRespawnPlayerKeepScore(t *testing.T) {
	gs := NewGameState()
	p := gs.AddPlayer("conn1")
	p.Score = 123
	oldStart := time.Now().Add(-5 * time.Minute)
	p.StartTime = oldStart

	gs.RespawnPlayerKeepScore(p.ID)
	if p.Score != 123 {
		t.Fatalf("KeepScore zerou o placar: %d", p.Score)
	}
	if !p.StartTime.Equal(oldStart) {
		t.Fatalf("KeepScore alterou StartTime: %v != %v", p.StartTime, oldStart)
	}

	gs.RespawnPlayer(p.ID)
	if p.Score != 0 {
		t.Fatalf("RespawnPlayer não zerou o placar: %d", p.Score)
	}
	if p.StartTime.Equal(oldStart) {
		t.Fatal("RespawnPlayer não renovou StartTime")
	}
}

// TestDoubleJump verifica que um jogador no ar com o pulo bufferizado executa
// o pulo duplo (1 pulo extra no ar).
func TestDoubleJump(t *testing.T) {
	gs := NewGameState()
	p := gs.AddPlayer("conn1")
	p.X = gs.ArenaWidth / 2
	p.Y = 200 // no ar, longe de qualquer tile
	p.IsOnGround = false
	p.VelocityX = 0
	p.VelocityY = 0
	p.JumpsUsed = 0
	p.jumpBufferedAt = time.Now()

	gs.ApplyPhysics()

	if p.JumpsUsed != 1 {
		t.Fatalf("JumpsUsed = %d, esperado 1 (pulo duplo)", p.JumpsUsed)
	}
	if p.VelocityY >= 0 {
		t.Fatalf("pulo duplo não aplicou impulso para cima (vy=%.2f)", p.VelocityY)
	}
}

// TestResolvePlayerCollisionsSeparates verifica que bolas sobrepostas são
// separadas e empurradas para longe (knockback base).
func TestResolvePlayerCollisionsSeparates(t *testing.T) {
	gs := NewGameState()
	a := gs.AddPlayer("conn1")
	b := gs.AddPlayer("conn2")
	a.X, a.Y = 0, 0
	b.X, b.Y = 10, 0 // sobreposição: dist 10 < raio 25+25
	a.VelocityX, a.VelocityY = 0, 0
	b.VelocityX, b.VelocityY = 0, 0

	gs.resolvePlayerCollisions(time.Now())

	dist := math.Hypot(b.X-a.X, b.Y-a.Y)
	if dist < 49.9 {
		t.Fatalf("bolas não separadas: dist=%.2f", dist)
	}
	if a.VelocityX >= 0 || b.VelocityX <= 0 {
		t.Fatalf("knockback errado: a.vx=%.2f b.vx=%.2f (esperado a<0 e b>0)", a.VelocityX, b.VelocityX)
	}
}

// TestResolvePlayerCollisionsDashKnockback verifica que o dash aplica
// knockback extra no oponente e recuo reduzido no atacante.
func TestResolvePlayerCollisionsDashKnockback(t *testing.T) {
	gs := NewGameState()
	a := gs.AddPlayer("conn1")
	b := gs.AddPlayer("conn2")
	a.X, a.Y = 0, 0
	b.X, b.Y = 10, 0
	a.VelocityX, a.VelocityY = 0, 0
	b.VelocityX, b.VelocityY = 0, 0
	now := time.Now()
	a.dashUntil = now.Add(DashDuration) // atacante em janela ativa

	gs.resolvePlayerCollisions(now)

	// Knockback do dash: b deve receber impulso extra além do base.
	if b.VelocityX <= KnockbackBase {
		t.Fatalf("dash não aplicou knockback extra em b (b.vx=%.2f)", b.VelocityX)
	}
	// Recuo do atacante é reduzido (DashRecoil), nunca igual ao do oponente.
	if a.VelocityX <= -KnockbackDash {
		t.Fatalf("atacante sofreu recuo pleno do dash (a.vx=%.2f)", a.VelocityX)
	}
}
