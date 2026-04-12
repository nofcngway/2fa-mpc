package twofaService

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vbncursed/vkr/twofa/internal/models"
)

// GetStatus returns the 2FA enrollment status for a user (per D-17).
// Returns nil, nil if user has no 2FA record (not set up).
func (s *TwoFAService) GetStatus(ctx context.Context, userID string) (*models.TwoFARecord, error) {
	record, err := s.storage.GetTwoFARecord(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get 2fa status: %w", err)
	}

	// Fire-and-forget audit event
	if err := s.eventProducer.PublishEvent(ctx, NewAuditEvent(userID, "2fa.status_checked", "success")); err != nil {
		slog.Warn("failed to publish audit event", "operation", "2fa.status_checked", "error", err)
	}

	return record, nil
}
