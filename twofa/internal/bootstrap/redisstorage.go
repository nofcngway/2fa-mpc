package bootstrap

import (
	"context"
	"log/slog"

	"github.com/vbncursed/vkr/twofa/config"
	"github.com/vbncursed/vkr/twofa/internal/services/twofaService"
	"github.com/vbncursed/vkr/twofa/internal/storage/redisstorage"
)

// NewSessionStorage creates a Redis-backed SessionStorage, falling back to
// NoOpSessionStorage when Redis is unavailable (rate limiting disabled, no panic).
func NewSessionStorage(ctx context.Context, cfg *config.Config) twofaService.SessionStorage {
	rs, err := redisstorage.NewRedisStorage(ctx, cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		slog.Warn("Redis unavailable, rate limiting disabled", "error", err)
		return &redisstorage.NoOpSessionStorage{}
	}
	return rs
}
