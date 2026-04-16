// Package redisstorage provides Redis-backed rate limiting and OTP counter storage.
package redisstorage

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vbncursed/vkr/twofa/internal/services/twofaService"
)

var _ twofaService.SessionStorage = (*RedisStorage)(nil)

// RedisStorage provides Redis persistence for the TwoFA service (rate limiting).
type RedisStorage struct {
	client *redis.Client
}

// NewRedisStorage creates a new RedisStorage and verifies the connection.
func NewRedisStorage(ctx context.Context, addr string, password string, db int) (*RedisStorage, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		DialTimeout:  5 * time.Second,
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
