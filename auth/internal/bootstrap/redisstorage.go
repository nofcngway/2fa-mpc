package bootstrap

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vbncursed/vkr/auth/config"
	"github.com/vbncursed/vkr/auth/internal/storage/redisstorage"
)

// NewRedisStorage creates a new Redis storage connection.
// Returns an error if Redis is unreachable — session operations require Redis.
func NewRedisStorage(ctx context.Context, cfg *config.Config) (*redisstorage.RedisStorage, error) {
	rs := redisstorage.New(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err := rs.Ping(ctx); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}
	slog.Info("Redis connected")
	return rs, nil
}
