package pgstorage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/vbncursed/vkr/auth/internal/domain"
)

// CreateUser inserts a new user into the database.
func (ps *PGStorage) CreateUser(ctx context.Context, user *domain.User) error {
	_, err := ps.pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`, user.ID, user.Email, user.PasswordHash, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrDuplicateEmail
		}
		return err
	}
	return nil
}

// GetUserByEmail retrieves a user by email address. Returns (nil, nil) if not found.
func (ps *PGStorage) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := ps.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}
