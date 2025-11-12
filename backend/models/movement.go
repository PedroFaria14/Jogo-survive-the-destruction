package models

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

// --- CONSTANTES DE JOGO (DEVE SER AS MESMAS DO FRONTEND) ---
const (
	ArenaWidth    = 800.0
	ArenaHeight   = 600.0
	TileSize      = 100.0 // Cada bloco da arena terá 100x100
	PlayerRadius  = 25.0  // Raio da bola no frontend

	// Constantes de Movimento e Física
	MoveSpeed     = 8.0
	JumpForce     = 15.0
	Gravity       = 1.5
	Friction      = 0.85
	AirResistance = 0.98
	BreakInterval = 5 // Segundos para quebrar um tile
)

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
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	VelocityX  float64 `json:"vx"`
	VelocityY  float64 `json:"vy"`
	IsOnGround bool    `json:"on_ground"`
	Health     int     `json:"health"`
	// NOVO: Campos de Placar
	IsDead    bool      `json:"is_dead"`
	Score     int       `json:"score"` // Placar em segundos
	StartTime time.Time `json:"-"`     // Não serializar, apenas para cálculo interno
}

type GameState struct {
	Players       map[string]*Player    `json:"players"`
	ArenaTiles    map[string]*ArenaTile `json:"arena_tiles"`
	Status        string                `json:"status"`
	lastBreakTime time.Time             // Controla o temporizador de destruição
}

type Command struct {
	Type  string `json:"type"`
	Right bool   `json:"right"`
	Left  bool   `json:"left"`
	Jump  bool   `json:"jump"`
}

func NewGameState() *GameState {
	status := &GameState{
		Players:       make(map[string]*Player),
		ArenaTiles:    make(map[string]*ArenaTile),
		Status:        "waiting",
		lastBreakTime: time.Now(),
	}
	// CHAMA A NOVA FUNÇÃO DE INICIALIZAÇÃO PERSONALIZADA
	status.initializeArenaCustom()
	rand.Seed(time.Now().UnixNano())
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

// initializeArenaCustom configura a arena de acordo com o design de plataformas
func (gs *GameState) initializeArenaCustom() {
	// A arena tem 8 colunas (0-7) e 6 linhas (0-5, onde 0 é o topo e 5 é a base)
	
	// LINHA 5 (Base/Chão)
	r5 := 5
	for c := 0; c < 8; c++ {
		gs.addTile(r5, c)
	}

	// LINHA 4 (Plataformas Intermediárias)
	r4 := 4
	// Plataforma 1: colunas 1, 2, 3
	for c := 1; c <= 3; c++ {
		gs.addTile(r4, c)
	}
	// Plataforma 2: colunas 5, 6, 7
	for c := 5; c <= 7; c++ {
		gs.addTile(r4, c)
	}

	// LINHA 3 (Plataformas Superiores)
	r3 := 3
	// Plataforma 3: colunas 0, 1
	for c := 0; c <= 1; c++ {
		gs.addTile(r3, c)
	}
	// Plataforma 4: colunas 3, 4
	for c := 3; c <= 4; c++ {
		gs.addTile(r3, c)
	}
	// Plataforma 5: colunas 6, 7
	for c := 6; c <= 7; c++ {
		gs.addTile(r3, c)
	}
	
	// LINHA 2 (Plataformas Mais Altas e quebradas)
	r2 := 2
	// Plataforma 6: coluna 0 (Canto L)
	gs.addTile(r2, 0)
	// Plataforma 7: coluna 5
	gs.addTile(r2, 5)
	// Plataforma 8: coluna 7
	gs.addTile(r2, 7)

	log.Printf("Arena inicializada com %d tiles no formato personalizado.", len(gs.ArenaTiles))
}

func (gs *GameState) initializeArena() {
    // Redireciona para a nova função personalizada
    gs.initializeArenaCustom()
}

func (gs *GameState) AddPlayer(connID string) *Player {
	id := fmt.Sprintf("player_%d", len(gs.Players)+1)
	player := &Player{
		ID:         id,
		ConnID:     connID,
		X:          ArenaWidth/2 + rand.Float64()*100 - 50, // Posição aleatória central
		Y:          ArenaHeight / 4,                        // Posição mais alta para cair
		VelocityX:  0,
		VelocityY:  0,
		IsOnGround: false,
		Health:     100,
		// NOVO: Inicializa o Placar
		IsDead:    false,
		Score:     0,
		StartTime: time.Now(),
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

// ProcessInput agora é um método do GameState e recebe o comando do Hub.
func (gs *GameState) ProcessInput(playerID string, cmd Command) {
	player, ok := gs.Players[playerID]
	if !ok || player.IsDead {
		return // Ignora se o jogador não for encontrado ou estiver morto
	}

	if cmd.Left {
		player.VelocityX = -MoveSpeed
	} else if cmd.Right {
		player.VelocityX = MoveSpeed
	} else {
		// A fricção já é aplicada em ApplyPhysics
	}

	if cmd.Jump && player.IsOnGround {
		player.VelocityY = -JumpForce
		player.IsOnGround = false
	}
}

func (gs *GameState) ApplyPhysics() {
	for _, player := range gs.Players {
		if player.IsDead {
			continue
		}

		// 1. Aplicar Forças e Resistência
		if player.IsOnGround {
			if player.VelocityX != 0 {
				player.VelocityX *= Friction
			}
		} else {
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
		player.VelocityY += Gravity
		player.Y += player.VelocityY

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
					playerHitTile = true
					break
				}
			}
		}

		// 5. Atualizar IsOnGround
		if !playerHitTile {
			player.IsOnGround = false
		}

		// 6. Morte por Queda (Caiu abaixo da tela)
		if player.Y > ArenaHeight+100 {
			player.IsDead = true
			log.Printf("Jogador %s atingiu a condição de morte.", player.ID)
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

	tile := activeTiles[rand.Intn(len(activeTiles))]
	tile.IsFalling = true
	gs.lastBreakTime = time.Now()
	log.Printf("Tile %s começou a cair", tile.ID)

	// Agenda a destruição final após 2 segundos (o tempo que o tile fica vermelho)
	time.AfterFunc(time.Second*2, func() {
		tile.IsActive = false
		tile.IsFalling = false
		log.Printf("Tile %s removido permanentemente. Buraco criado!", tile.ID)
	})
}
