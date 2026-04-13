package twofaService

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"golang.org/x/crypto/bcrypt"
)

// ErrInvalidBackupCode indicates that the backup code didn't match any unused code.
var ErrInvalidBackupCode = errors.New("2fa: invalid backup code")

// VerifyBackupCode validates a plaintext backup code against stored bcrypt hashes.
// On match, the code is marked as used (one-time use). Rate limiting applies.
func (s *TwoFAService) VerifyBackupCode(ctx context.Context, userID, code string) error {
	rows, err := s.storage.GetUnusedBackupCodeHashes(ctx, userID)
	if err != nil {
		return fmt.Errorf("get backup codes: %w", err)
	}

	for _, row := range rows {
		if err := bcrypt.CompareHashAndPassword([]byte(row.CodeHash), []byte(code)); err == nil {
			if err := s.storage.MarkBackupCodeUsed(ctx, row.ID); err != nil {
				return fmt.Errorf("mark backup code used: %w", err)
			}

			slog.Info("backup code verified", "user_id", userID)

			if err := s.eventProducer.PublishEvent(ctx, NewAuditEvent(userID, "2fa.backup_code_used", "success")); err != nil {
				slog.Warn("failed to publish audit event", "operation", "2fa.backup_code_used", "error", err)
			}

			return nil
		}
	}

	return ErrInvalidBackupCode
}
