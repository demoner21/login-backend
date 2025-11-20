package database

import (
	"database/sql"
	"fmt"

	_ "github.com/marcboeker/go-duckdb"
)

func NewDuckDB(path string) (*sql.DB, error) {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir DuckDB: %w", err)
	}

	return db, nil
}
