package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

// Score representa um registro de placar para exibição
type Score struct {
	PlayerID string `json:"-"` // ID interno; não exposto publicamente
	Name     string `json:"name"`
	ScoreSec int    `json:"score_seconds"`
}

// ScoreService define a interface para interagir com o placar
type ScoreService interface {
	GetTopScores() ([]Score, error)
	// SaveScore recebe o ID do jogador, o nome, o placar em segundos e a duração total.
	SaveScore(playerID, playerName string, scoreSeconds int, duration time.Duration) error
	// Close libera a conexão com o banco de dados.
	Close() error
}

// PostgresScoreService é a implementação que usa o PostgreSQL
type PostgresScoreService struct {
	DB *sql.DB
}

// initSchema garante que a tabela 'scores' exista com os campos necessários.
func (s *PostgresScoreService) initSchema() error {
	const createTableSQL = `
	CREATE TABLE IF NOT EXISTS scores (
		id SERIAL PRIMARY KEY,
		player_id VARCHAR(50) NOT NULL,
		player_name VARCHAR(50) NOT NULL,
		score_seconds INTEGER NOT NULL,
		duration_ms BIGINT NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := s.DB.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("falha ao criar ou verificar a tabela scores: %w", err)
	}
	log.Println("Tabela 'scores' verificada/criada com sucesso.")
	return nil
}

// NewPostgresScoreService inicializa o serviço e testa a conexão.
func NewPostgresScoreService(connStr string) (ScoreService, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir conexão com o DB: %w", err)
	}

	// Limites do pool: o Neon free tier tem ~20 conexões máximas. Configurar
	// tetos explícitos evita estourar o limite sob picos e goroutines de save.
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err = db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("erro ao conectar com o PostgreSQL: %w", err)
	}

	service := &PostgresScoreService{DB: db}

	if err := service.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return service, nil
}

// Close fecha a conexão com o banco de dados.
func (s *PostgresScoreService) Close() error {
	return s.DB.Close()
}

// GetTopScores busca os 10 melhores placares do banco de dados
func (s *PostgresScoreService) GetTopScores() ([]Score, error) {
	query := `SELECT player_id, player_name, score_seconds FROM scores ORDER BY score_seconds DESC LIMIT 10`

	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("erro ao executar query de placares: %w", err)
	}
	defer rows.Close()

	// Inicializa como slice vazio (não nil) para o JSON serializar [] e não null.
	scores := make([]Score, 0)
	for rows.Next() {
		var score Score
		// Assumindo que Name é o mesmo que PlayerID para este MVP
		if err := rows.Scan(&score.PlayerID, &score.Name, &score.ScoreSec); err != nil {
			log.Printf("Erro ao escanear linha: %v", err)
			continue
		}
		scores = append(scores, score)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro após iterar pelas linhas: %w", err)
	}

	return scores, nil
}

// SaveScore salva o placar final do jogador no banco de dados, com timeout
// de contexto e uma única tentativa de retry em caso de falha transitória.
func (s *PostgresScoreService) SaveScore(playerID, playerName string, scoreSeconds int, duration time.Duration) error {
	durationMs := duration.Milliseconds()

	// Prepara a query de inserção.
	query := `INSERT INTO scores (player_id, player_name, score_seconds, duration_ms) VALUES ($1, $2, $3, $4)`

	var err error
	for attempt := 0; attempt < 2; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err = s.DB.ExecContext(ctx, query, playerID, playerName, scoreSeconds, durationMs)
		cancel()
		if err == nil {
			log.Printf("Placar salvo: %s, Score: %d segundos", playerName, scoreSeconds)
			return nil
		}
	}
	return fmt.Errorf("erro ao salvar placar para %s: %w", playerID, err)
}
