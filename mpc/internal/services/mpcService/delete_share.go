package mpcService

import (
	"context"
	"fmt"
	"log/slog"
)

// DeleteShare deletes all shares for a user from this node.
// Returns the number of deleted shares. Returns 0 with no error if none exist (idempotent, per D-08).
func (s *MPCService) DeleteShare(ctx context.Context, userID string) (int64, error) {
	count, err := s.storage.DeleteSharesByUserID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("delete shares: %w", err)
	}

	// Fire-and-forget audit event
	if err := s.eventProducer.PublishEvent(ctx, NewAuditEvent(userID, "share.deleted", "success", s.nodeID)); err != nil {
		slog.Warn("failed to publish audit event", "operation", "share.deleted", "error", err)
	}

	return count, nil
}
