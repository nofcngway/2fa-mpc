package pgstorage

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PGStorage provides PostgreSQL data access for the Auth service.
type PGStorage struct {
	pool *pgxpool.Pool
}

// New creates a new PGStorage, connects to the database, and initializes tables.
func New(ctx context.Context, dsn string) (*PGStorage, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	storage := &PGStorage{pool: pool}
	if err := storage.initTables(ctx); err != nil {
		pool.Close()
		return nil, err
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
