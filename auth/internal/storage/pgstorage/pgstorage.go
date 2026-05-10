// Package pgstorage implements PostgreSQL-backed persistence for the Auth service.
package pgstorage

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PGStorage provides PostgreSQL data access for the Auth service.
type PGStorage struct {
	pool *pgxpool.Pool
}

// New creates a new PGStorage, connects to the database, and initializes tables.
func New(ctx context.Context, dsn string) (*PGStorage, error) {
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	// Sized to ≈4× available cores so concurrent bcrypt/JWT workers do not
	// queue on a connection. MinConns keeps a small warm pool so the first
	// requests after idle do not pay the dial latency.
	poolCfg.MaxConns = int32(max(8, runtime.NumCPU()*4))
	poolCfg.MinConns = int32(max(2, runtime.NumCPU()/2))
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

// initTables creates the required database tables if they do not exist.
func (ps *PGStorage) initTables(ctx context.Context) error {
	_, err := ps.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);
	`)
	return err
}

// Close releases the database connection pool.
func (ps *PGStorage) Close() {
	ps.pool.Close()
}
