package pgstorage

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/vbncursed/vkr/twofa/internal/domain"
)

// StoreBatchBackupCodes inserts bcrypt-hashed backup codes for a user.
// Each code gets a unique UUID as primary key. Uses a transaction for atomicity.
func (ps *PGStorage) StoreBatchBackupCodes(ctx context.Context, userID string, codeHashes []string) error {
	query := `INSERT INTO backup_codes (id, user_id, code_hash, is_used) VALUES ($1, $2, $3, FALSE)`

	tx, err := ps.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, hash := range codeHashes {
		id := uuid.New().String()
		_, err := tx.Exec(ctx, query, id, userID, hash)
		if err != nil {
			return fmt.Errorf("insert backup code: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit backup codes: %w", err)
	}
	return nil
}

// GetUnusedBackupCodeHashes returns all unused backup code hashes for a user.
func (ps *PGStorage) GetUnusedBackupCodeHashes(ctx context.Context, userID string) ([]domain.BackupCodeRow, error) {
	query := `SELECT id, code_hash FROM backup_codes WHERE user_id = $1 AND is_used = FALSE`
	rows, err := ps.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query backup codes: %w", err)
	}
	defer rows.Close()

	var codes []domain.BackupCodeRow
	for rows.Next() {
		var row domain.BackupCodeRow
		if err := rows.Scan(&row.ID, &row.CodeHash); err != nil {
			return nil, fmt.Errorf("scan backup code: %w", err)
		}
		codes = append(codes, row)
	}
	return codes, rows.Err()
}

// MarkBackupCodeUsed marks a single backup code as used by its ID.
func (ps *PGStorage) MarkBackupCodeUsed(ctx context.Context, codeID string) error {
	_, err := ps.pool.Exec(ctx, `UPDATE backup_codes SET is_used = TRUE WHERE id = $1`, codeID)
	if err != nil {
		return fmt.Errorf("mark backup code used: %w", err)
	}
	return nil
}
