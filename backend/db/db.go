package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

// ConnectDB estabelece a conexão com o banco de dados PostgreSQL.
// A string de conexão vem da variável de ambiente DATABASE_URL
// (definida no .env ou no ambiente do sistema), nunca de valor hardcoded.
func ConnectDB() (*sql.DB, error) {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		return nil, fmt.Errorf("a variável de ambiente DATABASE_URL não está definida")
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	log.Println("Conexão com o PostgreSQL estabelecida com sucesso!")
	return db, nil
}

// InitializeDatabase garante que a tabela 'scores' exista.
// O schema aqui deve permanecer idêntico ao usado pela camada de serviço
// (service/poster.go) para evitar divergência entre os dois pacotes.
func InitializeDatabase(db *sql.DB) error {
	const createTableSQL = `
	CREATE TABLE IF NOT EXISTS scores (
		id SERIAL PRIMARY KEY,
		player_id VARCHAR(50) NOT NULL,
		player_name VARCHAR(50) NOT NULL,
		score_seconds INTEGER NOT NULL,
		duration_ms BIGINT NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(createTableSQL)
	if err != nil {
		return err
	}
	log.Println("Tabela 'scores' verificada/criada com sucesso.")
	return nil
}