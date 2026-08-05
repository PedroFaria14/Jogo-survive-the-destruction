package models

import (
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"
)

// --- CONSTANTES DE JOGO (DEVE SER AS MESMAS DO FRONTEND) ---
const (
	ArenaWidth   = 800.0
	ArenaHeight  = 1000.0 // 10 linhas: dá abismo real abaixo das ilhas flutuantes
	TileSize     = 100.0  // Cada bloco da arena terá 100x100
	PlayerRadius = 25.0  // Raio base da bola no frontend (buffs podem mudar)

	// Constantes de Movimento e Física
	MoveSpeed          = 8.0
	JumpForce          = 20.0 // ~133px de altura: suficiente para subir blocos de 100px
	Gravity            = 1.5
	VariableJumpFactor = 0.6 // Fator de gravidade enquanto segura o pulo (pulo variável)
	Friction           = 0.85
	AirResistance      = 0.98

	// Polimento de controle
	CoyoteTime = 120 * time.Millisecond // Janela para pular após sair da borda
	JumpBuffer = 120 * time.Millisecond // Janela para pular após pressionar antes de pousar

	// Escape de buracos e anti-trava
	MaxAirJumps  = 1               // Pulo duplo: um pulo extra no ar para sair de buracos
	StuckTimeout = 4 * time.Second // Se parado no chão por muito tempo, respawna

	// Destruição da arena (quanto menor, mais frenético)
	BreakInterval = 3               // Segundos entre cada tile destruído
	FallDelay     = 1 * time.Second // Tempo que o tile fica vermelho antes de cair

	// Quadrado perdido: plataforma extra que aparece de vez em quando em
	// posição aleatória e permanece até ser destruída pelo timer normal.
	LostTileIntervalMin = 8 * time.Second
	LostTileIntervalMax = 14 * time.Second

	// Stocks (vidas por rodada)
	MaxLives = 3 // Vidas iniciais de cada jogador na rodada

	// Mecânica de empurrar (knockback)
	Restitution   = 0.85 // Elasticidade da colisão bola-bola
	KnockbackBase = 2.0  // Empurrão mínimo ao encostar em outro jogador
	KnockbackDash = 11.0 // Empurrão extra aplicado pelo Dash no oponente
	DashRecoil    = 0.3  // Recuo reduzido do atacante no Dash
	MaxSpeed      = 34.0 // Teto de velocidade (evita voar para fora da tela)

	// Geração procedural de ilhas (aumente para "escalar" o jogo)
	MinIslands     = 3  // Número mínimo de ilhas por rodada
	MaxIslands     = 7  // Número máximo de ilhas por rodada
	IslandWidthMin = 1  // Largura mínima de cada ilha (em tiles) — pilares arriscados
	IslandWidthMax = 6  // Largura máxima de cada ilha (em tiles) — bases seguras
	MaxGapCols     = 3  // Gap horizontal máximo entre ilhas (em tiles)
	MaxRiseTiles   = 2  // Subida máxima entre ilhas (em tiles)
	MaxArenaCols   = 26 // Teto de colunas da arena (define a largura)
	EdgeMarginCols = 1  // Margem mínima de abismo nas bordas laterais
	LayoutAttempts = 40 // Tentativas de layout antes de usar o fallback

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

	// Parâmetros da geração procedural de ilhas
	MinIslands     int `json:"min_islands"`
	MaxIslands     int `json:"max_islands"`
	IslandWidthMin int `json:"island_width_min"`
	IslandWidthMax int `json:"island_width_max"`
	MaxGapCols     int `json:"max_gap_cols"`

	// Parâmetros dos power-ups
	DropInterval  float64 `json:"powerup_interval"`  // Segundos entre drops
	DropLifetime  float64 `json:"powerup_lifetime"`  // Segundos até o drop desparecer
	RedDuration   float64 `json:"red_duration"`      // Duração do buff Tanque
	PurpleDuration float64 `json:"purple_duration"`  // Duração do buff Velocista
	BlueDuration  float64 `json:"blue_duration"`     // Duração do buff Planar
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
		MinIslands:    MinIslands,
		MaxIslands:    MaxIslands,
		IslandWidthMin: IslandWidthMin,
		IslandWidthMax: IslandWidthMax,
		MaxGapCols:    MaxGapCols,
		DropInterval:  DropInterval.Seconds(),
		DropLifetime:  DropLifetime.Seconds(),
		RedDuration:   BuffRedDuration.Seconds(),
		PurpleDuration: BuffPurpleDuration.Seconds(),
		BlueDuration:  BuffBlueDuration.Seconds(),
	}
}

// --- STRUCTS DO ESTADO DE JOGO ---

type ArenaTile struct {
	ID        string    `json:"id"`
	X         float64   `json:"x"`
	Y         float64   `json:"y"`
	IsFalling bool      `json:"is_falling"`
	IsActive  bool      `json:"is_active"`
	Kind      string    `json:"kind"` // Autotile: "top" | "mid" | "bottom"
	FallAt    time.Time `json:"-"`    // Quando o tile em queda é removido (serializado apenas pelo hub)
}

type Player struct {
	ID         string  `json:"id"`
	ConnID     string  `json:"conn_id"`
	Name       string  `json:"name"`
	Color      string  `json:"color"` // Hex #RRGGBB da bolinha ("" = cor padrão)
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	VelocityX  float64 `json:"vx"`
	VelocityY  float64 `json:"vy"`
	IsOnGround bool    `json:"on_ground"`
	// SpawnIndex é o índice de spawn fixado na criação do jogador. Mantê-lo
	// evita que o respawn dependa da ordem (aleatória) de iteração do mapa.
	SpawnIndex int `json:"-"`
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

	// Power-ups (buff ativo — apenas um por vez)
	Buff          string    `json:"buff"`           // Tipo do buff ativo ("" se nenhum)
	BuffUntil     time.Time `json:"-"`              // Fim do buff
	BuffRemaining float64   `json:"buff_remaining"` // Segundos restantes (HUD)

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
	PowerUps   map[string]*PowerUp   `json:"power_ups"`
	Status     string                `json:"status"`
	Round      int                   `json:"round"`      // Número da rodada atual
	RoundOver  bool                  `json:"round_over"` // Rodada encerrada, aguardando reinício
	Countdown  int                   `json:"countdown"`  // Segundos restantes para nova rodada
	ArenaWidth float64               `json:"arena_width"`
	ArenaHeight float64              `json:"arena_height"`

	// Power-ups
	DropCountdown float64   `json:"drop_countdown"` // Segundos até o próximo drop (HUD)
	nextDropAt    time.Time `json:"-"`
	nextPowerUpID int       `json:"-"`

	SpawnPoints []SpawnPoint `json:"-"` // Pontos de spawn da rodada atual

	lastBreakTime time.Time // Controla o temporizador de destruição
	nextLostTileAt time.Time // Quando o próximo quadrado perdido aparece
	nextPlayerID  int       // Contador monotônico para gerar IDs únicos
}

// SpawnPoint é uma posição segura de respawn acima da ilha central.
type SpawnPoint struct {
	X float64
	Y float64
}

type Command struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Right bool   `json:"right"`
	Left  bool   `json:"left"`
	Jump  bool   `json:"jump"`
	Dash  bool   `json:"dash"`
}

// SanitizeColor valida uma cor de bolinha enviada pelo cliente. Aceita apenas
// o formato "#RRGGBB" (hex). Retorna "" (cor padrão no frontend) se inválida.
func SanitizeColor(s string) string {
	s = strings.TrimSpace(s)
	if len(s) != 7 || s[0] != '#' {
		return ""
	}
	if _, err := strconv.ParseUint(s[1:], 16, 32); err != nil {
		return ""
	}
	return strings.ToLower(s)
}

func NewGameState() *GameState {
	status := &GameState{
		Players:       make(map[string]*Player),
		ArenaTiles:    make(map[string]*ArenaTile),
		PowerUps:      make(map[string]*PowerUp),
		Status:        "waiting",
		Round:         1,
		lastBreakTime: time.Now(),
		nextLostTileAt: time.Now().Add(LostTileIntervalMin),
		nextDropAt:    time.Now().Add(DropInterval),
		DropCountdown: DropInterval.Seconds(),
	}
	status.generateArena()
	return status
}

// addTileKind cria e adiciona um tile marcado com o tipo de autotile
// ("top" = superfície de grama, "mid" = miolo de terra, "bottom" = ponta).
func (gs *GameState) addTileKind(r, c int, kind string) {
	id := fmt.Sprintf("tile_%d_%d", r, c)
	gs.ArenaTiles[id] = &ArenaTile{
		ID:       id,
		X:        float64(c) * TileSize,
		Y:        float64(r) * TileSize,
		IsActive: true,
		Kind:     kind,
	}
}

// islandPlan descreve uma ilha gerada antes de ser materializada no grid.
type islandPlan struct {
	Col     int   // Coluna da borda esquerda
	Width   int   // Largura em tiles
	Profile []int // Altura (em tiles) por coluna
	Bottom  int   // Linha do tile mais baixo
}

// minTop retorna a linha do ponto mais alto da ilha.
func (ip *islandPlan) minTop() int {
	top := ip.Bottom - ip.Profile[0] + 1
	for _, h := range ip.Profile {
		if r := ip.Bottom - h + 1; r < top {
			top = r
		}
	}
	return top
}

// islandProfile monta o perfil em arco (centro alto, pontas baixas):
// ex.: largura 5 → [1,2,3,2,1].
func islandProfile(width int) []int {
	prof := make([]int, width)
	for k := 0; k < width; k++ {
		d := k
		if width-1-k < d {
			d = width - 1 - k
		}
		prof[k] = 1 + d
	}
	return prof
}

// imin retorna o menor entre dois inteiros.
func imin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// layoutResult é o resultado de um layout: ilhas e maior coluna ocupada.
type layoutResult struct {
	islands []islandPlan
	maxCol  int
}

// canReach aplica a "regra de ouro" do level design: a distância horizontal e
// vertical entre as bordas de duas ilhas deve ser percorrível com o pulo do
// Glowie (1 pulo + pulo duplo). É a trava que impede ilhas inalcançáveis.
func canReach(a, b islandPlan) bool {
	gap := 0
	switch {
	case b.Col >= a.Col+a.Width:
		gap = b.Col - (a.Col + a.Width)
	case a.Col >= b.Col+b.Width:
		gap = a.Col - (b.Col + b.Width)
	}
	rise := a.minTop() - b.minTop() // >0: b está mais alto
	switch {
	case rise <= 0:
		return gap <= MaxGapCols+1
	case rise <= 1:
		return gap <= MaxGapCols
	case rise <= MaxRiseTiles:
		return gap <= 1
	default:
		return false
	}
}

// buildLayout sorteia a distribuição de ilhas ao redor de uma ilha central de
// spawn. A alcançabilidade é validada A CADA ilha colocada (cada nova ilha
// deve ser pulável a partir da última do seu lado), o que garante por indução
// que todas são alcançáveis a partir do spawn. Um BFS final é mantido como
// trava de segurança ("regra de ouro").
func buildLayout() (layoutResult, bool) {
	n := MinIslands + rand.IntN(MaxIslands-MinIslands+1)
	occupied := map[string]bool{}
	islands := []islandPlan{}
	maxCol := 0

	place := func(col, width, bottom int) (islandPlan, bool) {
		prof := islandProfile(width)
		ip := islandPlan{Col: col, Width: width, Profile: prof, Bottom: bottom}
		for k := 0; k < width; k++ {
			tr := bottom - prof[k] + 1
			for r := tr; r <= bottom; r++ {
				if occupied[fmt.Sprintf("%d_%d", col+k, r)] {
					return ip, false
				}
			}
		}
		for k := 0; k < width; k++ {
			tr := bottom - prof[k] + 1
			for r := tr; r <= bottom; r++ {
				occupied[fmt.Sprintf("%d_%d", col+k, r)] = true
			}
		}
		islands = append(islands, ip)
		if col+width-1 > maxCol {
			maxCol = col + width - 1
		}
		return ip, true
	}

	// Ilha de spawn: um pouco maior, central, garantida e flutuando em altura
	// média-alta (bottom 4), deixando abismo visível abaixo.
	sw := imin(IslandWidthMin+2, IslandWidthMax)
	spawn, ok := place(MaxArenaCols/2-1, sw, 4)
	if !ok {
		return layoutResult{}, false
	}
	lastLeft, lastRight := spawn, spawn
	left, right := spawn.Col-1, spawn.Col+spawn.Width

	// Expande para os lados, alternando e validando alcançabilidade local.
	// A largura de cada ilha é limitada ao espaço disponível no lado, para
	// preencher a arena e chegar mais perto do total sorteado (min/max).
	for i := 0; i < n-1; i++ {
		placed := false
		order := []int{i % 2, (i + 1) % 2} // 0 = esquerda, 1 = direita
		for _, side := range order {
			last := lastLeft
			if side == 1 {
				last = lastRight
			}
			gap := 1 + rand.IntN(MaxGapCols)
			width := IslandWidthMin + rand.IntN(IslandWidthMax-IslandWidthMin+1)
			col := left - gap - width + 1
			if side == 1 {
				col = right + gap
			}
			// Limita a largura ao espaço útil do lado (garante margem).
			if side == 0 {
				if col < EdgeMarginCols {
					continue
				}
				if maxW := col - EdgeMarginCols + 1; width > maxW {
					width = maxW
				}
			} else {
				if col+IslandWidthMin-1+EdgeMarginCols > MaxArenaCols {
					continue
				}
				if maxW := MaxArenaCols - EdgeMarginCols - col + 1; width > maxW {
					width = maxW
				}
			}
			if width < IslandWidthMin {
				continue
			}
			// Alturas candidatas da base (bottom) da ilha, do alto (1) ao
			// baixo (6). A ordem é sorteada a cada colocação (rand.Perm) para
			// garantir variedade vertical — ilhas altas, médias e baixas —
			// respeitando a regra de alcançabilidade (canReach).
			bottomCandidates := []int{1, 2, 3, 4, 5, 6}
			for _, idx := range rand.Perm(len(bottomCandidates)) {
				bottom := bottomCandidates[idx]
				ip := islandPlan{Col: col, Width: width, Profile: islandProfile(width), Bottom: bottom}
				if !canReach(last, ip) {
					continue
				}
				if _, okk := place(col, width, bottom); !okk {
					continue
				}
				if side == 0 {
					lastLeft, left = ip, col-1
				} else {
					lastRight, right = ip, col+width
				}
				placed = true
				break
			}
			if placed {
				break
			}
		}
		if !placed {
			break
		}
	}

	if len(islands) < 2 {
		return layoutResult{}, false
	}

	// Trava matemática (segurança): BFS a partir da ilha de spawn (islands[0]).
	visited := make([]bool, len(islands))
	visited[0] = true
	queue := []int{0}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for j := range islands {
			if visited[j] {
				continue
			}
			if canReach(islands[cur], islands[j]) || canReach(islands[j], islands[cur]) {
				visited[j] = true
				queue = append(queue, j)
			}
		}
	}
	for _, v := range visited {
		if !v {
			return layoutResult{}, false
		}
	}

	return layoutResult{islands: islands, maxCol: maxCol}, true
}

// materialize converte as ilhas planejadas em tiles com autotile.
func (gs *GameState) materialize(res layoutResult) {
	gs.ArenaTiles = make(map[string]*ArenaTile)
	for _, ip := range res.islands {
		for k := 0; k < ip.Width; k++ {
			tr := ip.Bottom - ip.Profile[k] + 1
			for r := tr; r <= ip.Bottom; r++ {
				kind := "mid"
				if r == tr {
					kind = "top"
				} else if r == ip.Bottom {
					kind = "bottom"
				}
				gs.addTileKind(r, ip.Col+k, kind)
			}
		}
	}
}

// computeSpawnPoints gera pontos de spawn acima da superfície da ilha central.
func (gs *GameState) computeSpawnPoints(sp islandPlan) {
	top := sp.minTop()
	gs.SpawnPoints = gs.SpawnPoints[:0]
	for k := 0; k < sp.Width; k++ {
		if sp.Bottom-sp.Profile[k]+1 == top {
			gs.SpawnPoints = append(gs.SpawnPoints, SpawnPoint{
				X: (float64(sp.Col+k) + 0.5) * TileSize,
				Y: float64(top)*TileSize - 40,
			})
		}
	}
}

// fallbackArena é um mapa mínimo determinístico (sempre alcançável), usado se
// o gerador falhar repetidamente.
func (gs *GameState) fallbackArena() {
	gs.ArenaTiles = make(map[string]*ArenaTile)
	gs.addTileKind(4, 1, "top")
	gs.addTileKind(5, 1, "bottom")
	gs.addTileKind(4, 2, "top")
	gs.addTileKind(5, 2, "bottom")
	gs.addTileKind(3, 3, "top")
	gs.addTileKind(4, 3, "mid")
	gs.addTileKind(5, 3, "bottom")
	gs.addTileKind(4, 5, "top")
	gs.addTileKind(5, 5, "bottom")
	gs.addTileKind(4, 6, "top")
	gs.addTileKind(5, 6, "bottom")
	gs.ArenaWidth = 8 * TileSize
	gs.ArenaHeight = ArenaHeight
	gs.SpawnPoints = []SpawnPoint{{X: 250, Y: 360}, {X: 350, Y: 360}}
}

// generateArena reconstrói a arena do zero com ilhas geradas proceduralmente.
// A cada rodada o mapa muda (posições, quantidades e tamanhos sorteados),
// respeitando a regra de alcançabilidade e dimensionando a largura da arena.
func (gs *GameState) generateArena() {
	for attempt := 0; attempt < LayoutAttempts; attempt++ {
		res, ok := buildLayout()
		if !ok {
			continue
		}
		gs.computeSpawnPoints(res.islands[0])
		gs.materialize(res)
		gs.ArenaWidth = float64(res.maxCol+EdgeMarginCols+1) * TileSize
		gs.ArenaHeight = ArenaHeight
		log.Printf("Arena procedural gerada: %d ilhas, %d tiles, largura %.0fpx.",
			len(res.islands), len(gs.ArenaTiles), gs.ArenaWidth)
		return
	}
	gs.fallbackArena()
	log.Printf("Arena procedural falhou após %d tentativas; usado fallback.", LayoutAttempts)
}

// spawnPosition retorna um ponto de spawn seguro acima da ilha central do
// mapa atual, rotacionando pelos pontos gerados de acordo com o índice.
func (gs *GameState) spawnPosition(index int) (float64, float64) {
	if len(gs.SpawnPoints) == 0 {
		return 150, 120
	}
	p := gs.SpawnPoints[index%len(gs.SpawnPoints)]
	return p.X, p.Y
}

func (gs *GameState) AddPlayer(connID string) *Player {
	// Contador monotônico: evita IDs duplicados quando jogadores saem e entram.
	gs.nextPlayerID++
	id := fmt.Sprintf("player_%d", gs.nextPlayerID)
	x, y := gs.spawnPosition(gs.nextPlayerID)
	player := &Player{
		ID:         id,
		ConnID:     connID,
		X:          x,
		Y:          y,
		VelocityX:  0,
		VelocityY:  0,
		IsOnGround: false,
		IsDead:     false,
		Score:      0,
		Lives:      MaxLives,
		StartTime:  time.Now(),
		JumpsUsed:  0,
		SpawnIndex: gs.nextPlayerID,
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
// A pontuação é zerada: representa "perda de vida" (killzone) ou início de
// rodada. Use RespawnPlayerKeepScore para teleportes que não são morte
// (ex.: anti-trava), onde o tempo sobrevivido deve ser preservado.
func (gs *GameState) RespawnPlayer(playerID string) {
	gs.respawnPlayer(playerID, true)
}

// RespawnPlayerKeepScore teleporta o jogador preservando Score e StartTime.
// Usado pelo anti-trava: o jogador não perdeu vida, então não deve perder o
// tempo acumulado de sobrevivência.
func (gs *GameState) RespawnPlayerKeepScore(playerID string) {
	gs.respawnPlayer(playerID, false)
}

func (gs *GameState) respawnPlayer(playerID string, resetScore bool) {
	p, ok := gs.Players[playerID]
	if !ok {
		return
	}
	x, y := gs.spawnPosition(p.SpawnIndex)
	p.X = x
	p.Y = y
	p.VelocityX = 0
	p.VelocityY = 0
	p.IsOnGround = false
	p.IsDead = false
	if resetScore {
		p.Score = 0
		p.StartTime = time.Now()
	}
	p.ScoreSaved = false
	p.lastGroundedAt = time.Time{}
	p.jumpBufferedAt = time.Time{}
	p.JumpsUsed = 0
	p.stuckSince = time.Time{}
	p.DashQueued = false
	p.dashHeld = false
	p.dashUntil = time.Time{}
	p.dashReadyAt = time.Time{}
	p.DashCd = 0
	p.Buff = ""
	p.BuffUntil = time.Time{}
	p.BuffRemaining = 0
	log.Printf("Jogador %s respawnado.", p.ID)
}

// ResetRound reconstrói a arena e respawna todos os jogadores conectados,
// iniciando uma nova rodada com vidas cheias.
func (gs *GameState) ResetRound() {
	gs.generateArena()
	gs.lastBreakTime = time.Now()
	gs.nextLostTileAt = time.Now().Add(LostTileIntervalMin)
	gs.clearPowerUps()
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

		// Raio e velocidade atuais (afetados por buffs de power-up).
		radius := player.radius(now)
		speed := MoveSpeed * player.speedMult()

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
			player.VelocityX = -speed
		} else if player.RightHeld {
			player.VelocityX = speed
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
				if player.Y+radius > tile.Y && player.Y-radius < tile.Y+TileSize {

					// Colisão com a esquerda do Tile (movendo para a direita)
					if player.VelocityX > 0 && player.X+radius > tile.X && player.X-radius < tile.X {
						player.X = tile.X - radius // Ajusta a posição para o limite do Tile
						player.VelocityX = 0
					}

					// Colisão com a direita do Tile (movendo para a esquerda)
					if player.VelocityX < 0 && player.X-radius < tile.X+TileSize && player.X+radius > tile.X+TileSize {
						player.X = tile.X + TileSize + radius // Ajusta a posição para o limite do Tile
						player.VelocityX = 0
					}
				}
			}
		}

		// 3. Limite da Arena (Paredes) - Colisão X final
		if player.X <= radius {
			player.X = radius
			player.VelocityX = 0
		}
		if player.X >= gs.ArenaWidth-radius {
			player.X = gs.ArenaWidth - radius
			player.VelocityX = 0
		}

		// --- 4. Tentar Atualizar Posição Y e Checar Colisão Vertical ---

		// Aplica a gravidade e o movimento vertical
		g := Gravity * player.gravityScale()
		if player.JumpHeld && player.VelocityY < 0 {
			// Pulo variável: segurar a tecla reduz a gravidade ao subir.
			g = Gravity * VariableJumpFactor * player.gravityScale()
		}
		player.VelocityY += g
		player.Y += player.VelocityY

		// Checagem de Colisão Vertical (Teto - subindo)
		if player.VelocityY < 0 {
			for _, tile := range gs.ArenaTiles {
				if tile.IsActive &&
					player.X > tile.X && player.X < tile.X+TileSize &&
					player.Y-radius < tile.Y+TileSize && player.Y-radius > tile.Y {

					player.Y = tile.Y + TileSize + radius
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
					player.Y+radius > tile.Y && player.Y+radius < tile.Y+TileSize {

					player.Y = tile.Y - radius
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
				// Teleporte anti-trava NÃO é morte: preserva Score/StartTime.
				gs.RespawnPlayerKeepScore(player.ID)
				continue
			}
		} else {
			player.stuckSince = time.Time{}
		}

		// 6. Queda no abismo (killzone): perde vida ou é eliminado.
		if player.Y > gs.ArenaHeight+100 {
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
	// Antes, aplica o teto de velocidade em todos os vivos: limita a queda
	// (evita tunneling pelos tiles) e o knockback, sem afetar o dash.
	for _, player := range gs.Players {
		if !player.IsDead {
			clampSpeed(player)
		}
	}
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
// as bolas e aplica impulso elástico ponderado pela massa (buff Tank faz o
// jogador mais pesado empurrar menos e empurrar mais o oponente) com
// restituição. Durante a janela do dash, o atacante aplica knockback extra no
// oponente e sofre recuo reduzido — a base da mecânica de empurrar.
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
			radiusA := a.radius(now)
			radiusB := b.radius(now)
			dx := b.X - a.X
			dy := b.Y - a.Y
			dist := math.Hypot(dx, dy)
			minDist := radiusA + radiusB

			if dist >= minDist || dist == 0 {
				continue
			}

			// Normal apontando de a para b.
			nx, ny := dx/dist, dy/dist

			// Separação: distribui a sobreposição proporcionalmente à massa
			// (o mais leve desloca mais).
			massA := a.mass()
			massB := b.mass()
			overlap := minDist - dist
			totalMass := massA + massB
			a.X -= nx * overlap * (massB / totalMass)
			a.Y -= ny * overlap * (massB / totalMass)
			b.X += nx * overlap * (massA / totalMass)
			b.Y += ny * overlap * (massA / totalMass)

			// Impulso elástico ponderado pela massa (restituição).
			rel := (b.VelocityX-a.VelocityX)*nx + (b.VelocityY-a.VelocityY)*ny
			if rel < 0 {
				imp := -(1 + Restitution) * rel * (massB / totalMass)
				a.VelocityX -= imp * nx
				a.VelocityY -= imp * ny
				b.VelocityX += imp * (massA / massB) * nx
				b.VelocityY += imp * (massA / massB) * ny
			}

			// Knockback mínimo garantido ao encostar (bumps sempre empurram),
			// ponderado pela massa: o pesado empurra mais e recua menos.
			b.VelocityX += nx * KnockbackBase * (massA / massB)
			b.VelocityY += ny * KnockbackBase * (massA / massB)
			a.VelocityX -= nx * KnockbackBase * (massB / massA)
			a.VelocityY -= ny * KnockbackBase * (massB / massA)

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
	now := time.Now()
	if now.Sub(gs.lastBreakTime).Seconds() < BreakInterval {
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

	tile := activeTiles[rand.IntN(len(activeTiles))]
	tile.IsFalling = true
	tile.FallAt = now.Add(FallDelay) // Tempo que o tile fica vermelho antes de cair
	gs.lastBreakTime = now
	log.Printf("Tile %s começou a cair", tile.ID)
}

// ExpireFallingTiles remove os tiles cuja janela de queda expirou. Deve ser
// chamado pela goroutine do Hub (serializado), assim como todo o resto do
// GameState — a destruição nunca acontece em goroutines externas.
func (gs *GameState) ExpireFallingTiles(now time.Time) {
	for _, tile := range gs.ArenaTiles {
		if tile.IsFalling && now.After(tile.FallAt) {
			tile.IsFalling = false
			tile.IsActive = false
			log.Printf("Tile %s removido permanentemente. Buraco criado!", tile.ID)
		}
	}
}

// SpawnLostTile cria, de vez em quando, um "quadrado perdido": um tile de
// plataforma extra em posição aleatória vazia da arena. Ele permanece no pool
// normal de destruição (fica até ser destruído pelo timer). Deve ser chamado
// pela goroutine do Hub (serializado).
func (gs *GameState) SpawnLostTile(now time.Time) {
	if now.Before(gs.nextLostTileAt) {
		return
	}

	cols := int(gs.ArenaWidth / TileSize)
	rows := int(gs.ArenaHeight / TileSize)
	if cols <= 0 || rows <= 0 {
		gs.nextLostTileAt = now.Add(LostTileIntervalMin)
		return
	}

	for attempt := 0; attempt < 20; attempt++ {
		r := rand.IntN(rows)
		c := rand.IntN(cols)
		id := fmt.Sprintf("tile_%d_%d", r, c)
		if _, exists := gs.ArenaTiles[id]; exists {
			continue
		}
		gs.ArenaTiles[id] = &ArenaTile{
			ID:        id,
			X:         float64(c) * TileSize,
			Y:         float64(r) * TileSize,
			IsActive:  true,
			Kind:      "lost",
		}
		interval := LostTileIntervalMin.Seconds() +
			rand.Float64()*(LostTileIntervalMax-LostTileIntervalMin).Seconds()
		gs.nextLostTileAt = now.Add(time.Duration(interval * float64(time.Second)))
		log.Printf("Quadrado perdido apareceu em %s", id)
		return
	}

	// Sem célula vazia: tenta de novo no próximo intervalo.
	gs.nextLostTileAt = now.Add(LostTileIntervalMin)
}
