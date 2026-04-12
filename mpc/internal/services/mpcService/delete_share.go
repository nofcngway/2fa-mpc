package mpcService

import (
	"context"
	"fmt"
)

// DeleteShare deletes all shares for a user from this node.
// Returns the number of deleted shares. Returns 0 with no error if none exist (idempotent, per D-08).
func (s *MPCService) DeleteShare(ctx context.Context, userID string) (int64, error) {
	count, err := s.storage.DeleteSharesByUserID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("delete shares: %w", err)
	}
	return count, nil
}
