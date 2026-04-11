package redisstorage

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

// RedisStorage provides Redis persistence for the TwoFA service (rate limiting).
type RedisStorage struct {
	client *redis.Client
}

// NewRedisStorage creates a new RedisStorage and verifies the connection.
func NewRedisStorage(ctx context.Context, addr string, password string, db int) (*RedisStorage, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	slog.Info("Redis connected", "service", "twofa")
	return &RedisStorage{client: client}, nil
}

// Close closes the Redis connection.
func (rs *RedisStorage) Close() error {
	return rs.client.Close()
}
