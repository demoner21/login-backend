package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	PostgresSSLMode  string
}

func Load() Config {
	// Carregar .env (ignora erro se arquivo não existir)
	if err := godotenv.Load(); err != nil {
		log.Printf("⚠️ Arquivo .env não encontrado, usando variáveis de ambiente do sistema")
	}

	return Config{
		PostgresHost:     getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:     getEnv("POSTGRES_PORT", "5432"),
		PostgresUser:     getEnv("POSTGRES_USER", "admin"),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", "password"),
		PostgresDB:       getEnv("POSTGRES_DB", "appdb"),
		PostgresSSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
	}
}

func (c Config) GetConnectionString() string {
	return "host=" + c.PostgresHost +
		" port=" + c.PostgresPort +
		" user=" + c.PostgresUser +
		" password=" + c.PostgresPassword +
		" dbname=" + c.PostgresDB +
		" sslmode=" + c.PostgresSSLMode
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
