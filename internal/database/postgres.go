package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func NewPostgres(connString string) (*sql.DB, error) {
	log.Printf("🔗 Conectando ao PostgreSQL: %s", connString)

	db, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, fmt.Errorf("erro ao conectar com PostgreSQL: %w", err)
	}

	// Testar a conexão
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("erro ao pingar PostgreSQL: %w", err)
	}

	log.Printf("✅ Conectado com sucesso ao PostgreSQL")
	return db, nil
}
