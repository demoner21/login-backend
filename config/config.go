package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DuckDBPath     string
	DuckDBDatabase string
}

func Load() Config {
	// Carregar .env (ignora erro se arquivo não existir)
	if err := godotenv.Load(); err != nil {
		log.Printf("⚠️ Arquivo .env não encontrado, usando variáveis de ambiente do sistema")
	}

	dbName := os.Getenv("DUCKDB_DATABASE")
	if dbName == "" {
		log.Fatal("❌ DUCKDB_DATABASE não configurado. Configure no arquivo .env ou variáveis de ambiente")
	}

	// Criar diretório data se não existir
	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Fatalf("❌ Erro ao criar diretório data: %v", err)
	}

	log.Printf("✅ Banco de dados: %s", dbName)

	return Config{
		DuckDBPath:     "./data/" + dbName,
		DuckDBDatabase: dbName,
	}
}
