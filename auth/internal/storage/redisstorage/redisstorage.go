package redisstorage

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// RedisStorage provides Redis data access for session and cache operations.
type RedisStorage struct {
	client *redis.Client
}

// New creates a new RedisStorage with the given connection parameters.
func New(addr, password string, db int) *RedisStorage {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &RedisStorage{client: client}
}

// Ping checks the Redis connection.
func (rs *RedisStorage) Ping(ctx context.Context) error {
	return rs.client.Ping(ctx).Err()
}

// Close releases the Redis connection.
func (rs *RedisStorage) Close() error {
	return rs.client.Close()
}
