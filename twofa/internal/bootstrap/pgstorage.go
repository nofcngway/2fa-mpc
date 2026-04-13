package bootstrap

import (
	"context"

	"github.com/vbncursed/vkr/twofa/config"
	"github.com/vbncursed/vkr/twofa/internal/storage/pgstorage"
)

// NewPGStorage creates a new PostgreSQL storage instance.
func NewPGStorage(ctx context.Context, cfg *config.Config) (*pgstorage.PGStorage, error) {
	return pgstorage.NewPGStorage(ctx, cfg.Database.DSN)
}
