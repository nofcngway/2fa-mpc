package mpcService

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/vbncursed/vkr/mpc/internal/domain"
)

// StoreShare encrypts share data and persists it.
// Returns the generated share ID.
func (s *MPCService) StoreShare(ctx context.Context, userID string, shareIndex int, shareData []byte) (string, error) {
	encryptedData, nonce, err := s.encrypt(shareData)
	if err != nil {
		return "", fmt.Errorf("encrypt share: %w", err)
	}

	share := &domain.Share{
		ID:            uuid.New().String(),
		UserID:        userID,
		ShareIndex:    shareIndex,
		EncryptedData: encryptedData,
		Nonce:         nonce,
		CreatedAt:     time.Now(),
	}

	if err := s.storage.CreateShare(ctx, share); err != nil {
		if errors.Is(err, domain.ErrDuplicateShare) {
			return "", domain.ErrDuplicateShare
		}
		return "", fmt.Errorf("store share: %w", err)
	}

	// Fire-and-forget audit event
	if err := s.eventProducer.PublishEvent(ctx, NewAuditEvent(userID, "share.stored", "success", s.nodeID)); err != nil {
		slog.Warn("failed to publish audit event", "operation", "share.stored", "error", err)
	}

	return share.ID, nil
}
