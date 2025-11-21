package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRedis(host, port, password string) (*redis.Client, error) {
	addr := fmt.Sprintf("%s:%s", host, port)
	log.Printf("🔗 Conectando ao Redis: %s", addr)

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0, // DB padrão
	})

	// Testar conexão (Ping)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("erro ao conectar com Redis: %w", err)
	}

	log.Printf("✅ Conectado com sucesso ao Redis")
	return rdb, nil
}
