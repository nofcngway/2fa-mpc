// Package pgstorage provides PostgreSQL storage for the MPC service.
package pgstorage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vbncursed/vkr/mpc/internal/services/mpcService"
)

var _ mpcService.Storage = (*PGStorage)(nil)

// PGStorage implements PostgreSQL-based share storage.
type PGStorage struct {
	pool *pgxpool.Pool
}

// New creates a new PGStorage instance, connects to the database, and initializes tables.
func New(ctx context.Context, dsn string) (*PGStorage, error) {
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

	storage := &PGStorage{pool: pool}
	if err := storage.initTables(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("init tables: %w", err)
	}

	return storage, nil
}

// initTables creates required database tables if they do not exist.
func (ps *PGStorage) initTables(ctx context.Context) error {
	_, err := ps.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS shares (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL,
			share_index INT NOT NULL,
			encrypted_data BYTEA NOT NULL,
			nonce BYTEA NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			UNIQUE(user_id, share_index)
		);
	`)
	return err
}

// Close closes the database connection pool.
func (ps *PGStorage) Close() {
	ps.pool.Close()
}
