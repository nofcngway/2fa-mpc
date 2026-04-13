package bootstrap

import (
	"context"
	"log/slog"

	"github.com/vbncursed/vkr/mpc/config"
	"github.com/vbncursed/vkr/mpc/internal/storage/pgstorage"
)

// NewPGStorage creates a new PostgreSQL storage instance.
func NewPGStorage(ctx context.Context, cfg *config.Config) (*pgstorage.PGStorage, error) {
	storage, err := pgstorage.New(ctx, cfg.Database.DSN)
	if err != nil {
		return nil, err
	}
	slog.Info("PostgreSQL connected")
	return storage, nil
}
