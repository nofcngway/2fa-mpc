// Package redisstorage implements Redis-backed session and cache storage for the Auth service.
package redisstorage

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStorage provides Redis data access for session and cache operations.
type RedisStorage struct {
	client *redis.Client
}

// New creates a new RedisStorage with the given connection parameters.
// Timeouts prevent indefinite blocking on unresponsive Redis instances.
func New(addr, password string, db int) *RedisStorage {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
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
