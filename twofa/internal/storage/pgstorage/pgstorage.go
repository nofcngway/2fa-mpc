// Package pgstorage provides PostgreSQL persistence for 2FA records and backup codes.
package pgstorage

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbncursed/vkr/twofa/internal/services/twofaService"
)

var _ twofaService.Storage = (*PGStorage)(nil)

// PGStorage provides PostgreSQL persistence for the TwoFA service.
type PGStorage struct {
	pool *pgxpool.Pool
}

// NewPGStorage creates a new PGStorage and initializes tables.
func NewPGStorage(ctx context.Context, dsn string) (*PGStorage, error) {
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	poolCfg.MaxConns = 25
	poolCfg.MinConns = 2
	poolCfg.MaxConnLifetime = 5 * time.Minute
	poolCfg.MaxConnIdleTime = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	ps := &PGStorage{pool: pool}
	if err := ps.initTables(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("init tables: %w", err)
	}

	slog.Info("PostgreSQL connected", "service", "twofa")
	return ps, nil
}

// Close closes the database connection pool.
func (ps *PGStorage) Close() {
	ps.pool.Close()
}

// initTables creates required database tables if they do not exist.
func (ps *PGStorage) initTables(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS twofa_records (
			user_id UUID PRIMARY KEY,
			is_enabled BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS backup_codes (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL,
			code_hash VARCHAR(255) NOT NULL,
			is_used BOOLEAN NOT NULL DEFAULT FALSE
		);

		CREATE INDEX IF NOT EXISTS idx_backup_codes_user_id ON backup_codes (user_id);
	`

	_, err := ps.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("create tables: %w", err)
	}

	return nil
}
