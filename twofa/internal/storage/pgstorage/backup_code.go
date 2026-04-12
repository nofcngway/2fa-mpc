package pgstorage

import (
	"context"
	"fmt"

	"github.com/google/uuid"
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
