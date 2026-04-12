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

// EnableTwoFA sets is_enabled=true for the user's 2FA record.
func (ps *PGStorage) EnableTwoFA(ctx context.Context, userID string) error {
	query := `UPDATE twofa_records SET is_enabled = TRUE WHERE user_id = $1`
	_, err := ps.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("enable twofa: %w", err)
	}
	return nil
}

// DeleteTwoFARecord removes the 2FA record for a user.
func (ps *PGStorage) DeleteTwoFARecord(ctx context.Context, userID string) error {
	query := `DELETE FROM twofa_records WHERE user_id = $1`
	_, err := ps.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("delete twofa record: %w", err)
	}
	return nil
}

// DeleteBackupCodes removes all backup codes for a user.
func (ps *PGStorage) DeleteBackupCodes(ctx context.Context, userID string) error {
	query := `DELETE FROM backup_codes WHERE user_id = $1`
	_, err := ps.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("delete backup codes: %w", err)
	}
	return nil
}
