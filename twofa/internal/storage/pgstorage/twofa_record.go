package pgstorage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/vbncursed/vkr/twofa/internal/models"
)

// CreateTwoFARecord inserts a new 2FA record with is_enabled=false.
func (ps *PGStorage) CreateTwoFARecord(ctx context.Context, userID string) error {
	query := `INSERT INTO twofa_records (user_id, is_enabled) VALUES ($1, FALSE)`
	_, err := ps.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("create twofa record: %w", err)
	}
	return nil
}

// GetTwoFARecord retrieves the 2FA record for a user. Returns nil, nil if not found.
func (ps *PGStorage) GetTwoFARecord(ctx context.Context, userID string) (*models.TwoFARecord, error) {
	query := `SELECT user_id, is_enabled, created_at FROM twofa_records WHERE user_id = $1`
	row := ps.pool.QueryRow(ctx, query, userID)

	var record models.TwoFARecord
	err := row.Scan(&record.UserID, &record.IsEnabled, &record.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get twofa record: %w", err)
	}
	return &record, nil
}
