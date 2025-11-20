package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/marcboeker/go-duckdb"
)

func NewDuckDB(path string) (*sql.DB, error) {
	log.Printf("🔗 Conectando ao DuckDB: %s", path)

	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir DuckDB: %w", err)
	}

	// Testar a conexão
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("erro ao conectar com DuckDB: %w", err)
	}

	log.Printf("✅ Conectado com sucesso ao DuckDB: %s", path)
	return db, nil
}
