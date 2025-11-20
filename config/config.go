package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DuckDBPath string
}

func Load() Config {
	godotenv.Load()

	dbName := os.Getenv("DUCKDB_DATABASE")
	if dbName == "" {
		log.Fatal("DUCKDB_DATABASE não configurado no .env")
	}

	return Config{
		DuckDBPath: "./data/" + dbName,
	}
}
