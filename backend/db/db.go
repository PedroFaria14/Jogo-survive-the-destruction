package db

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

// ConnectDB estabelece a conexão com o banco de dados PostgreSQL.
func ConnectDB() (*sql.DB, error) {
	// A string de conexão deve corresponder à sua configuração local do PostgreSQL.
	// user=postgres password=1234567 dbname=survive_the_destruction host=localhost sslmode=disable
	connStr := "user=postgres password=1234567 dbname=survive_the_destruction host=localhost sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	log.Println("Conexão com o PostgreSQL estabelecida com sucesso!")
	return db, nil
}

// O próximo passo seria criar uma função para garantir que a tabela 'scores' exista.
// Isso é opcional, mas recomendado.
func InitializeDatabase(db *sql.DB) error {
	const createTableSQL = `
	CREATE TABLE IF NOT EXISTS scores (
		id SERIAL PRIMARY KEY,
		player_id VARCHAR(50) NOT NULL,
		player_name VARCHAR(50) NOT NULL,
		score INTEGER NOT NULL,
		duration_ms INTEGER NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(createTableSQL)
	if err != nil {
		return err
	}
	log.Println("Tabela 'scores' verificada/criada com sucesso.")
	return nil
}
