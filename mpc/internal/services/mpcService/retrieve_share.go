package mpcService

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/vbncursed/vkr/mpc/internal/domain"
)

// RetrieveShare retrieves and decrypts a share by user_id and share_index.
func (s *MPCService) RetrieveShare(ctx context.Context, userID string, shareIndex int) ([]byte, error) {
	share, err := s.storage.GetShare(ctx, userID, shareIndex)
	if err != nil {
		if errors.Is(err, domain.ErrShareNotFound) {
			return nil, domain.ErrShareNotFound
		}
		return nil, fmt.Errorf("get share: %w", err)
	}

	plaintext, err := s.decrypt(share.EncryptedData, share.Nonce)
	if err != nil {
		return nil, fmt.Errorf("decrypt share (user_id=%s, share_index=%d, node_id=%d): %w", userID, shareIndex, s.nodeID, err)
	}

	// Fire-and-forget audit event
	if err := s.eventProducer.PublishEvent(ctx, NewAuditEvent(userID, "share.retrieved", "success", s.nodeID)); err != nil {
		slog.Warn("failed to publish audit event", "operation", "share.retrieved", "error", err)
	}

	return plaintext, nil
}
