package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisLimitCounter implementa httprate.LimitCounter (assinaturas compatíveis com v0.15.0)
type RedisLimitCounter struct {
	client *redis.Client
	window time.Duration
	prefix string
}

func NewRedisLimitCounter(client *redis.Client, prefix string) *RedisLimitCounter {
	return &RedisLimitCounter{
		client: client,
		prefix: prefix,
	}
}

// Config guarda a janela (chamado pelo httprate)
func (r *RedisLimitCounter) Config(requestLimit int, windowLength time.Duration) {
	r.window = windowLength
}

// Increment: apenas incrementa atomica e retorna erro (assinatura exigida)
func (r *RedisLimitCounter) Increment(key string, currentWindow time.Time) error {
	fullKey := r.makeKey(key, currentWindow)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Script: INCR e seta EXPIRE caso current==1 (atomic)
	script := `
		local current = redis.call("INCR", KEYS[1])
		if current == 1 then
			redis.call("EXPIRE", KEYS[1], ARGV[1])
		end
		return current
	`

	ttlSec := int(r.window.Seconds())
	cmd := r.client.Eval(ctx, script, []string{fullKey}, ttlSec)
	if cmd.Err() != nil {
		return cmd.Err()
	}
	// não precisamos do valor retornado aqui (o httprate faz o Get separadamente)
	return nil
}

// IncrementBy: incrementa por "amount" (usado internamente por httprate em alguns casos)
func (r *RedisLimitCounter) IncrementBy(key string, currentWindow time.Time, amount int) error {
	fullKey := r.makeKey(key, currentWindow)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// script usando INCRBY e set EXPIRE se for a primeira vez
	script := `
		local current = redis.call("INCRBY", KEYS[1], ARGV[1])
		if tonumber(ARGV[1]) > 0 and current == tonumber(ARGV[1]) then
			redis.call("EXPIRE", KEYS[1], ARGV[2])
		end
		return current
	`

	ttlSec := int(r.window.Seconds())
	cmd := r.client.Eval(ctx, script, []string{fullKey}, amount, ttlSec)
	if cmd.Err() != nil {
		return cmd.Err()
	}
	return nil
}

// Get: retorna contagem atual (na janela corrente) e ttlSeconds
// assinatura: Get(key, currentWindow, previousWindow time.Time) (count int, ttlSeconds int, err error)
func (r *RedisLimitCounter) Get(key string, currentWindow, previousWindow time.Time) (int, int, error) {
	fullKey := r.makeKey(key, currentWindow)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	val, err := r.client.Get(ctx, fullKey).Int()
	if err != nil {
		if err == redis.Nil {
			// chave não existe -> zero
			return 0, int(r.window.Seconds()), nil
		}
		return 0, 0, err
	}

	return val, int(r.window.Seconds()), nil
}

func (r *RedisLimitCounter) makeKey(key string, window time.Time) string {
	return fmt.Sprintf("%s:%s:%d", r.prefix, key, window.Unix())
}
