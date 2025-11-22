package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	PostgresSSLMode  string
	RedisHost        string
	RedisPort        string
	RedisPassword    string
	JWTSecret        string
	AllowedOrigins   []string

	AdminEmail    string
	AdminPassword string

	UploadProvider string
	UploadDir      string
	AppURL         string
}

func Load() *Config {
	// Tenta carregar .env, mas não falha se não existir (pode ser variáveis de sistema)
	_ = godotenv.Load()

	cfg := &Config{
		PostgresHost:     os.Getenv("POSTGRES_HOST"),
		PostgresPort:     os.Getenv("POSTGRES_PORT"),
		PostgresUser:     os.Getenv("POSTGRES_USER"),
		PostgresPassword: os.Getenv("POSTGRES_PASSWORD"),
		PostgresDB:       os.Getenv("POSTGRES_DB"),
		PostgresSSLMode:  os.Getenv("POSTGRES_SSLMODE"),
		RedisHost:        os.Getenv("REDIS_HOST"),
		RedisPort:        os.Getenv("REDIS_PORT"),
		RedisPassword:    os.Getenv("REDIS_PASSWORD"),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		AllowedOrigins:   strings.Split(os.Getenv("ALLOWED_ORIGINS"), ","),

		AdminEmail:    os.Getenv("ADMIN_EMAIL"),
		AdminPassword: os.Getenv("ADMIN_PASSWORD"),

		UploadProvider: os.Getenv("UPLOAD_PROVIDER"),
		UploadDir:      os.Getenv("UPLOAD_DIR"),
		AppURL:         os.Getenv("APP_URL"),
	}

	// Validação Crítica: Se faltar segredo, a aplicação NÃO SOBE.
	if cfg.JWTSecret == "" {
		log.Fatal("❌ FATAL: JWT_SECRET não está configurado.")
	}
	if cfg.PostgresPassword == "" {
		log.Fatal("❌ FATAL: POSTGRES_PASSWORD não está configurado.")
	}
	// Adicione validações para os outros campos de banco se desejar

	return cfg
}

func (c *Config) GetConnectionString() string {
	return "host=" + c.PostgresHost +
		" port=" + c.PostgresPort +
		" user=" + c.PostgresUser +
		" password=" + c.PostgresPassword +
		" dbname=" + c.PostgresDB +
		" sslmode=" + c.PostgresSSLMode
}
