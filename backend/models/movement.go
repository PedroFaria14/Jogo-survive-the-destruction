package models

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"
)

// rng é uma fonte randômica local (rng.Seed não é chamado), seguro para uso
// pelo processo. O uso de rand.New + rand.NewSource evita o uso de rand.Seed
// global (deprecado desde Go 1.20).
var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// --- CONSTANTES DE JOGO (DEVE SER AS MESMAS DO FRONTEND) ---
const (
	ArenaWidth   = 800.0
	ArenaHeight  = 600.0
	TileSize     = 100.0 // Cada bloco da arena terá 100x100
	PlayerRadius = 25.0  // Raio da bola no frontend

	// Constantes de Movimento e Física
	MoveSpeed          = 8.0
	JumpForce          = 20.0 // ~133px de altura: suficiente para subir blocos de 100px
	Gravity            = 1.5
	VariableJumpFactor = 0.6 // Fator de gravidade enquanto segura o pulo (pulo variável)
	Friction           = 0.85
	AirResistance      = 0.98
	BreakInterval      = 5 // Segundos para quebrar um tile

	// Polimento de controle
	CoyoteTime = 120 * time.Millisecond // Janela para pular após sair da borda
	JumpBuffer = 120 * time.Millisecond // Janela para pular após pressionar antes de pousar

	// Escape de buracos e anti-trava
	MaxAirJumps  = 1               // Pulo duplo: um pulo extra no ar para sair de buracos
	StuckTimeout = 4 * time.Second // Se parado no chão por muito tempo, respawna

	// Stocks (vidas por rodada)
	MaxLives = 3 // Vidas iniciais de cada jogador na rodada

	// Mecânica de empurrar (knockback)
	Restitution   = 0.85 // Elasticidade da colisão bola-bola
	KnockbackBase = 2.0  // Empurrão mínimo ao encostar em outro jogador
	KnockbackDash = 11.0 // Empurrão extra aplicado pelo Dash no oponente
	DashRecoil    = 0.3  // Recuo reduzido do atacante no Dash
	MaxSpeed      = 34.0 // Teto de velocidade (evita voar para fora da tela)

	// Dash
	DashSpeed    = 20.0
	DashCooldown = 1500 * time.Millisecond // Tempo entre dashes
	DashDuration = 250 * time.Millisecond  // Janela ativa do dash (hitbox de knockback)
)

// GameConfig reúne as constantes de jogo compartilhadas com o frontend
// (expostas via GET /api/config para evitar duplicação e drift).
type GameConfig struct {
	ArenaWidth    float64 `json:"arena_width"`
	ArenaHeight   float64 `json:"arena_height"`
	TileSize      float64 `json:"tile_size"`
	PlayerRadius  float64 `json:"player_radius"`
	MoveSpeed     float64 `json:"move_speed"`
	JumpForce     float64 `json:"jump_force"`
	Gravity       float64 `json:"gravity"`
	BreakInterval int     `json:"break_interval"`
	MaxAirJumps   int     `json:"max_air_jumps"`
	MaxLives      int     `json:"max_lives"`
	DashSpeed     float64 `json:"dash_speed"`
	DashCooldown  float64 `json:"dash_cooldown"`
	DashDuration  float64 `json:"dash_duration"`
	KnockbackDash float64 `json:"knockback_dash"`
	Restitution   float64 `json:"restitution"`
}

// GetGameConfig retorna as constantes de jogo para o frontend.
func GetGameConfig() GameConfig {
	return GameConfig{
		ArenaWidth:    ArenaWidth,
		ArenaHeight:   ArenaHeight,
		TileSize:      TileSize,
		PlayerRadius:  PlayerRadius,
		MoveSpeed:     MoveSpeed,
		JumpForce:     JumpForce,
		Gravity:       Gravity,
		BreakInterval: BreakInterval,
		MaxAirJumps:   MaxAirJumps,
		MaxLives:      MaxLives,
		DashSpeed:     DashSpeed,
		DashCooldown:  DashCooldown.Seconds(),
		DashDuration:  DashDuration.Seconds(),
		KnockbackDash: KnockbackDash,
		Restitution:   Restitution,
	}
}

// --- STRUCTS DO ESTADO DE JOGO ---

type ArenaTile struct {
	ID        string  `json:"id"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	IsFalling bool    `json:"is_falling"`
	IsActive  bool    `json:"is_active"`
}

type Player struct {
	ID         string  `json:"id"`
	ConnID     string  `json:"conn_id"`
	Name       string  `json:"name"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	VelocityX  float64 `json:"vx"`
	VelocityY  float64 `json:"vy"`
	IsOnGround bool    `json:"on_ground"`
	Health     int     `json:"health"`
	// Campos de Placar
	IsDead     bool      `json:"is_dead"`
	Score      int       `json:"score"` // Placar em segundos
	StartTime  time.Time `json:"-"`     // Não serializar, apenas para cálculo interno
	ScoreSaved bool      `json:"-"`     // Evita salvar o placar mais de uma vez

	// Stocks
	Lives int `json:"lives"` // Vidas restantes na rodada

	// Dash
	DashCd      float64   `json:"dash_cd"` // Segundos restantes do cooldown (HUD)
	DashQueued  bool      `json:"-"`
	dashHeld    bool      `json:"-"`
	dashUntil   time.Time `json:"-"` // Fim da janela ativa do dash
	dashReadyAt time.Time `json:"-"` // Quando o próximo dash será liberado

	// Campos de controle
	lastGroundedAt time.Time `json:"-"`          // Último momento em que estava no chão (coyote time)
	jumpBufferedAt time.Time `json:"-"`          // Momento em que o pulo foi pressionado (jump buffer)
	JumpsUsed      int       `json:"jumps_used"` // Pulos usados no ar (pulo duplo)
	JumpHeld       bool      `json:"-"`          // Tecla de pulo pressionada (pulo variável)
	LeftHeld       bool      `json:"-"`          // Tecla de esquerda pressionada (input contínuo)
	RightHeld      bool      `json:"-"`          // Tecla de direita pressionada (input contínuo)
	stuckSince     time.Time `json:"-"`          // Quando começou a ficar parado no chão (anti-trava)
}

type GameState struct {
	Players    map[string]*Player    `json:"players"`
	ArenaTiles map[string]*ArenaTile `json:"arena_tiles"`
	Status     string                `json:"status"`
	Round      int                   `json:"round"`      // Número da rodada atual
	RoundOver  bool                  `json:"round_over"` // Rodada encerrada, aguardando reinício
	Countdown  int                   `json:"countdown"`  // Segundos restantes para nova rodada

	lastBreakTime time.Time // Controla o temporizador de destruição
	nextPlayerID  int       // Contador monotônico para gerar IDs únicos
	roundGen      int       // Geração da rodada (invalida callbacks de tiles antigos)
}

type Command struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Right bool   `json:"right"`
	Left  bool   `json:"left"`
	Jump  bool   `json:"jump"`
	Dash  bool   `json:"dash"`
}

func NewGameState() *GameState {
	status := &GameState{
		Players:       make(map[string]*Player),
		ArenaTiles:    make(map[string]*ArenaTile),
		Status:        "waiting",
		Round:         1,
		lastBreakTime: time.Now(),
	}
	// CHAMA A NOVA FUNÇÃO DE INICIALIZAÇÃO PERSONALIZADA
	status.initializeArenaCustom()
	return status
}

// addTile é uma função auxiliar para criar e adicionar um tile no GameState
func (gs *GameState) addTile(r, c int) {
	id := fmt.Sprintf("tile_%d_%d", r, c)
	tile := &ArenaTile{
		ID:        id,
		X:         float64(c) * TileSize,
		Y:         float64(r) * TileSize,
		IsFalling: false,
		IsActive:  true,
	}
	gs.ArenaTiles[id] = tile
}

// initializeArenaCustom configura o mapa em formato de ilhas com abismo
// central, pirâmides escalonadas nas pontas e uma ponte flutuante sobre o
// abismo (área de alto risco). O fundo do mapa é vazio: quem cai morre.
func (gs *GameState) initializeArenaCustom() {
	// A arena tem 8 colunas (0-7) e 6 linhas (0-5, onde 0 é o topo e 5 é a base).
	// Layout:
	// r1 (y100):  X · · · · · · X        plataformas altas dos cantos
	// r2 (y200):  X · · X X · X ·        ponte central flutuante (c3,c4)
	// r3 (y300):  · X X · · X X ·        degraus da subida escalonada
	// r4 (y400):  · X X · · X X ·
	// r5 (y500):  X X X · · X X X        duas ilhas-base + abismo 200px central

	// Pirâmide esquerda
	gs.addTile(5, 0)
	gs.addTile(5, 1)
	gs.addTile(5, 2)
	gs.addTile(4, 1)
	gs.addTile(4, 2)
	gs.addTile(3, 1)
	gs.addTile(3, 2)
	gs.addTile(2, 1)

	// Pirâmide direita (espelhada)
	gs.addTile(5, 5)
	gs.addTile(5, 6)
	gs.addTile(5, 7)
	gs.addTile(4, 5)
	gs.addTile(4, 6)
	gs.addTile(3, 5)
	gs.addTile(3, 6)
	gs.addTile(2, 6)

	// Pontes/plataformas
	gs.addTile(2, 3) // Ponte central flutuante sobre o abismo
	gs.addTile(2, 4)
	gs.addTile(1, 0) // Canto alto esquerdo
	gs.addTile(1, 7) // Canto alto direito

	log.Printf("Arena inicializada com %d tiles no formato de ilhas.", len(gs.ArenaTiles))
}

// spawnPosition retorna um ponto de spawn seguro acima das ilhas, alternando
// entre as duas pirâmides de acordo com a paridade do índice.
func spawnPosition(index int) (float64, float64) {
	if index%2 == 0 {
		return 150, 120
	}
	return 650, 120
}

func (gs *GameState) AddPlayer(connID string) *Player {
	// Contador monotônico: evita IDs duplicados quando jogadores saem e entram.
	gs.nextPlayerID++
	id := fmt.Sprintf("player_%d", gs.nextPlayerID)
	x, y := spawnPosition(gs.nextPlayerID)
	player := &Player{
		ID:         id,
		ConnID:     connID,
		X:          x,
		Y:          y,
		VelocityX:  0,
		VelocityY:  0,
		IsOnGround: false,
		Health:     100,
		IsDead:     false,
		Score:      0,
		Lives:      MaxLives,
		StartTime:  time.Now(),
		JumpsUsed:  0,
	}
	gs.Players[id] = player
	log.Printf("Jogador %s adicionado na posição (%.2f, %.2f)", id, player.X, player.Y)
	return player
}

func (gs *GameState) RemovePlayer(playerID string) {
	if _, exists := gs.Players[playerID]; exists {
		delete(gs.Players, playerID)
		log.Printf("Jogador %s removido do GameState", playerID)
	}
}

// RespawnPlayer reinicia o jogador na mesma arena (mesmo mapa), mantendo o ID
// para que o cliente continue identificado. Preserva o campo Lives (quem
// chama decide se deve zerar as vidas — ex.: ResetRound ou comando restart).
func (gs *GameState) RespawnPlayer(playerID string) {
	p, ok := gs.Players[playerID]
	if !ok {
		return
	}
	index := 0
	for _, other := range gs.Players {
		if other.ID == playerID {
			break
		}
		index++
	}
	x, y := spawnPosition(index)
	p.X = x
	p.Y = y
	p.VelocityX = 0
	p.VelocityY = 0
	p.IsOnGround = false
	p.Health = 100
	p.IsDead = false
	p.Score = 0
	p.ScoreSaved = false
	p.StartTime = time.Now()
	p.lastGroundedAt = time.Time{}
	p.jumpBufferedAt = time.Time{}
	p.JumpsUsed = 0
	p.stuckSince = time.Time{}
	p.DashQueued = false
	p.dashHeld = false
	p.dashUntil = time.Time{}
	p.dashReadyAt = time.Time{}
	p.DashCd = 0
	log.Printf("Jogador %s respawnado.", p.ID)
}

// ResetRound reconstrói a arena e respawna todos os jogadores conectados,
// iniciando uma nova rodada com vidas cheias.
func (gs *GameState) ResetRound() {
	gs.roundGen++
	gs.ArenaTiles = make(map[string]*ArenaTile)
	gs.initializeArenaCustom()
	gs.lastBreakTime = time.Now()
	gs.RoundOver = false
	gs.Countdown = 0
	gs.Round++
	gs.Status = "playing"
	for _, p := range gs.Players {
		p.Lives = MaxLives
		gs.RespawnPlayer(p.ID)
	}
	log.Println("Rodada resetada: arena reconstruída e jogadores respawnados.")
}

// NoActiveTiles indica se não restam blocos ativos na arena.
func (gs *GameState) NoActiveTiles() bool {
	for _, t := range gs.ArenaTiles {
		if t.IsActive {
			return false
		}
	}
	return true
}

// ProcessInput agora é um método do GameState e recebe o comando do Hub.
func (gs *GameState) ProcessInput(playerID string, cmd Command) {
	player, ok := gs.Players[playerID]
	if !ok || player.IsDead {
		return // Ignora se o jogador não for encontrado ou estiver morto
	}

	// Input contínuo: guarda o estado das teclas. O ApplyPhysics aplica a
	// velocidade enquanto a tecla estiver pressionada (sem depender de repetição).
	player.LeftHeld = cmd.Left
	player.RightHeld = cmd.Right

	// Jump buffer: registra o pressionamento; o pulo é executado no pouso
	// (ou ainda dentro da janela de coyote time) pelo ApplyPhysics.
	if cmd.Jump {
		player.jumpBufferedAt = time.Now()
	}
	// Pulo variável: mantém o estado da tecla para reduzir a gravidade ao subir.
	player.JumpHeld = cmd.Jump

	// Dash: edge-detect — só dispara no pressionamento, não enquanto segurado.
	if cmd.Dash && !player.dashHeld && !player.IsDead {
		if time.Now().After(player.dashReadyAt) {
			player.DashQueued = true
		}
	}
	player.dashHeld = cmd.Dash
}

func (gs *GameState) ApplyPhysics() {
	now := time.Now()

	for _, player := range gs.Players {
		if player.IsDead {
			continue
		}

		// Atualiza o cooldown do dash para o HUD (segundos restantes).
		player.DashCd = math.Max(0, time.Until(player.dashReadyAt).Seconds())

		// 0. Processa o dash agendado (edge-detect feito no ProcessInput).
		if player.DashQueued {
			facing := 1.0
			if player.LeftHeld {
				facing = -1
			} else if player.RightHeld {
				facing = 1
			} else if player.VelocityX != 0 {
				facing = math.Copysign(1, player.VelocityX)
			}
			player.VelocityX = facing * DashSpeed
			player.dashUntil = now.Add(DashDuration)
			player.dashReadyAt = now.Add(DashCooldown)
			player.DashQueued = false
		}

		prevX := player.X // Para detecção de movimento (anti-trava)

		// 0. Tentativa de pulo (jump buffer + coyote time + pulo duplo)
		if now.Sub(player.jumpBufferedAt) <= JumpBuffer {
			onGround := player.IsOnGround || now.Sub(player.lastGroundedAt) <= CoyoteTime
			if onGround {
				player.VelocityY = -JumpForce
				player.IsOnGround = false
				player.lastGroundedAt = time.Time{}
				player.jumpBufferedAt = time.Time{} // Consome o buffer
				player.JumpsUsed = 0
			} else if player.JumpsUsed < MaxAirJumps {
				// Pulo duplo: permite um pulo extra no ar para sair de buracos.
				player.VelocityY = -JumpForce
				player.JumpsUsed++
				player.jumpBufferedAt = time.Time{}
			}
		}

		// 1. Aplicar Forças e Resistência
		if player.dashing(now) {
			// Rajada do dash: mantém a velocidade do dash sem recontrole
			// direcional nem resistência do ar durante a janela ativa.
		} else if player.LeftHeld {
			player.VelocityX = -MoveSpeed
		} else if player.RightHeld {
			player.VelocityX = MoveSpeed
		} else if player.IsOnGround {
			player.VelocityX *= Friction
		}
		if !player.IsOnGround && !player.dashing(now) {
			player.VelocityX *= AirResistance
		}

		// --- 2. Tentar Atualizar Posição X e Checar Colisão Horizontal ---

		// Aplica o movimento horizontal
		player.X += player.VelocityX

		// Checagem de Colisão Horizontal
		for _, tile := range gs.ArenaTiles {
			if tile.IsActive {
				// Verifica se o Player Y está dentro da faixa vertical do Tile
				// Permite a colisão lateral se o jogador estiver verticalmente alinhado com o tile
				if player.Y+PlayerRadius > tile.Y && player.Y-PlayerRadius < tile.Y+TileSize {

					// Colisão com a esquerda do Tile (movendo para a direita)
					if player.VelocityX > 0 && player.X+PlayerRadius > tile.X && player.X-PlayerRadius < tile.X {
						player.X = tile.X - PlayerRadius // Ajusta a posição para o limite do Tile
						player.VelocityX = 0
					}

					// Colisão com a direita do Tile (movendo para a esquerda)
					if player.VelocityX < 0 && player.X-PlayerRadius < tile.X+TileSize && player.X+PlayerRadius > tile.X+TileSize {
						player.X = tile.X + TileSize + PlayerRadius // Ajusta a posição para o limite do Tile
						player.VelocityX = 0
					}
				}
			}
		}

		// 3. Limite da Arena (Paredes) - Colisão X final
		if player.X <= PlayerRadius {
			player.X = PlayerRadius
			player.VelocityX = 0
		}
		if player.X >= ArenaWidth-PlayerRadius {
			player.X = ArenaWidth - PlayerRadius
			player.VelocityX = 0
		}

		// --- 4. Tentar Atualizar Posição Y e Checar Colisão Vertical ---

		// Aplica a gravidade e o movimento vertical
		g := Gravity
		if player.JumpHeld && player.VelocityY < 0 {
			// Pulo variável: segurar a tecla reduz a gravidade ao subir.
			g = Gravity * VariableJumpFactor
		}
		player.VelocityY += g
		player.Y += player.VelocityY

		// Checagem de Colisão Vertical (Teto - subindo)
		if player.VelocityY < 0 {
			for _, tile := range gs.ArenaTiles {
				if tile.IsActive &&
					player.X > tile.X && player.X < tile.X+TileSize &&
					player.Y-PlayerRadius < tile.Y+TileSize && player.Y-PlayerRadius > tile.Y {

					player.Y = tile.Y + TileSize + PlayerRadius
					player.VelocityY = 0
					break
				}
			}
		}

		// Checagem de Colisão Vertical (Pouso)
		playerHitTile := false
		for _, tile := range gs.ArenaTiles {
			if tile.IsActive {
				// Colisão Vertical (Topo do tile)
				if player.VelocityY > 0 &&
					player.X > tile.X && player.X < tile.X+TileSize &&
					player.Y+PlayerRadius > tile.Y && player.Y+PlayerRadius < tile.Y+TileSize {

					player.Y = tile.Y - PlayerRadius
					player.VelocityY = 0
					player.IsOnGround = true
					player.lastGroundedAt = now // Renova o coyote time
					player.JumpsUsed = 0        // Pousou: pulo duplo é resetado
					playerHitTile = true
					break
				}
			}
		}

		// 5. Atualizar IsOnGround
		if !playerHitTile {
			player.IsOnGround = false
		}

		// 5.5 Anti-trava: jogador vivo e parado no chão por muito tempo é
		// respawnado (evita ficar encurralado em buracos sem saída).
		dx := player.X - prevX
		if player.IsOnGround {
			if math.Abs(dx) > 1.0 {
				player.stuckSince = time.Time{}
			} else if player.stuckSince.IsZero() {
				player.stuckSince = now
			} else if now.Sub(player.stuckSince) > StuckTimeout {
				gs.RespawnPlayer(player.ID)
				continue
			}
		} else {
			player.stuckSince = time.Time{}
		}

		// 6. Queda no abismo (killzone): perde vida ou é eliminado.
		if player.Y > ArenaHeight+100 {
			if player.Lives > 1 {
				player.Lives--
				log.Printf("Jogador %s caiu no abismo. Vidas restantes: %d", player.ID, player.Lives)
				gs.RespawnPlayer(player.ID)
			} else {
				player.IsDead = true
				log.Printf("Jogador %s atingiu a condição de morte.", player.ID)
			}
		}
	}

	// 7. Colisão bola-bola com knockback (empurrão por velocidade relativa).
	gs.resolvePlayerCollisions(now)
}

// clampSpeed limita o módulo da velocidade do jogador (evita voar para fora).
func clampSpeed(p *Player) {
	speed := math.Hypot(p.VelocityX, p.VelocityY)
	if speed > MaxSpeed {
		factor := MaxSpeed / speed
		p.VelocityX *= factor
		p.VelocityY *= factor
	}
}

// dashing indica se o jogador está dentro da janela ativa do dash.
func (p *Player) dashing(now time.Time) bool {
	return now.Before(p.dashUntil)
}

// resolvePlayerCollisions detecta sobreposição entre jogadores vivos, separa
// as bolas e aplica impulso elástico (massa igual) com restituição. Durante a
// janela do dash, o atacante aplica knockback extra no oponente e sofre recuo
// reduzido — a base da mecânica de empurrar.
func (gs *GameState) resolvePlayerCollisions(now time.Time) {
	alive := make([]*Player, 0, len(gs.Players))
	for _, p := range gs.Players {
		if !p.IsDead {
			alive = append(alive, p)
		}
	}

	for i := 0; i < len(alive); i++ {
		for j := i + 1; j < len(alive); j++ {
			a, b := alive[i], alive[j]
			dx := b.X - a.X
			dy := b.Y - a.Y
			dist := math.Hypot(dx, dy)
			minDist := 2 * PlayerRadius

			if dist >= minDist || dist == 0 {
				continue
			}

			// Normal apontando de a para b.
			nx, ny := dx/dist, dy/dist

			// Separação: distribui a sobreposição igualmente.
			overlap := minDist - dist
			a.X -= nx * overlap / 2
			a.Y -= ny * overlap / 2
			b.X += nx * overlap / 2
			b.Y += ny * overlap / 2

			// Impulso elástico de massa igual (restituição).
			rel := (b.VelocityX-a.VelocityX)*nx + (b.VelocityY-a.VelocityY)*ny
			if rel < 0 {
				imp := -(1 + Restitution) * rel / 2
				a.VelocityX -= imp * nx
				a.VelocityY -= imp * ny
				b.VelocityX += imp * nx
				b.VelocityY += imp * ny
			}

			// Knockback mínimo garantido ao encostar (bumps sempre empurram).
			a.VelocityX -= nx * KnockbackBase
			a.VelocityY -= ny * KnockbackBase
			b.VelocityX += nx * KnockbackBase
			b.VelocityY += ny * KnockbackBase

			// Dash: atacante em janela ativa aplica knockback extra no oponente
			// e sofre recuo reduzido.
			if a.dashing(now) {
				b.VelocityX += nx * KnockbackDash
				b.VelocityY += ny * KnockbackDash
				a.VelocityX -= nx * KnockbackDash * DashRecoil
				a.VelocityY -= ny * KnockbackDash * DashRecoil
			}
			if b.dashing(now) {
				a.VelocityX -= nx * KnockbackDash
				a.VelocityY -= ny * KnockbackDash
				b.VelocityX += nx * KnockbackDash * DashRecoil
				b.VelocityY += ny * KnockbackDash * DashRecoil
			}

			clampSpeed(a)
			clampSpeed(b)
		}
	}
}

func (gs *GameState) CheckArenaDestruction() {
	if time.Since(gs.lastBreakTime).Seconds() < BreakInterval {
		return
	}

	activeTiles := []*ArenaTile{}
	for _, tile := range gs.ArenaTiles {
		if tile.IsActive && !tile.IsFalling {
			activeTiles = append(activeTiles, tile)
		}
	}
	if len(activeTiles) == 0 {
		return
	}

	gen := gs.roundGen // Geração atual: invalida callbacks de rodadas anteriores
	tile := activeTiles[rng.Intn(len(activeTiles))]
	tile.IsFalling = true
	gs.lastBreakTime = time.Now()
	log.Printf("Tile %s começou a cair", tile.ID)

	// Agenda a destruição final após 2 segundos (o tempo que o tile fica vermelho)
	time.AfterFunc(time.Second*2, func() {
		if gs.roundGen != gen {
			return // Rodada mudou; este tile não pertence mais à arena atual
		}
		tile.IsActive = false
		tile.IsFalling = false
		log.Printf("Tile %s removido permanentemente. Buraco criado!", tile.ID)
	})
}
