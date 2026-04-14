package pgstorage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/vbncursed/vkr/mpc/internal/domain"
)

// CreateShare inserts a new encrypted share into PostgreSQL.
func (ps *PGStorage) CreateShare(ctx context.Context, share *domain.Share) error {
	_, err := ps.pool.Exec(ctx,
		`INSERT INTO shares (id, user_id, share_index, encrypted_data, nonce, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		share.ID, share.UserID, share.ShareIndex,
		share.EncryptedData, share.Nonce, share.CreatedAt,
	)
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return domain.ErrDuplicateShare
		}
		return fmt.Errorf("create share: %w", err)
	}
	return nil
}

// GetShare retrieves a single share by user_id and share_index.
func (ps *PGStorage) GetShare(ctx context.Context, userID string, shareIndex int) (*domain.Share, error) {
	row := ps.pool.QueryRow(ctx,
		`SELECT id, user_id, share_index, encrypted_data, nonce, created_at
		 FROM shares WHERE user_id = $1 AND share_index = $2`,
		userID, shareIndex,
	)
	var s domain.Share
	err := row.Scan(&s.ID, &s.UserID, &s.ShareIndex, &s.EncryptedData, &s.Nonce, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrShareNotFound
		}
		return nil, fmt.Errorf("get share: %w", err)
	}
	return &s, nil
}

// DeleteSharesByUserID deletes all shares for a given user_id.
func (ps *PGStorage) DeleteSharesByUserID(ctx context.Context, userID string) (int64, error) {
	tag, err := ps.pool.Exec(ctx,
		`DELETE FROM shares WHERE user_id = $1`, userID,
	)
	if err != nil {
		return 0, fmt.Errorf("delete shares: %w", err)
	}
	return tag.RowsAffected(), nil
}
