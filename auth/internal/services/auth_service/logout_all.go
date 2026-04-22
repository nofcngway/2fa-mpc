package auth_service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vbncursed/vkr/auth/internal/publisher"
)

// LogoutAll revokes all sessions for a user by deleting all their tokens from Redis.
func (s *AuthService) LogoutAll(ctx context.Context, userID string) error {
	if err := s.sessionStorage.DeleteAllUserTokens(ctx, userID); err != nil {
		return fmt.Errorf("delete all user tokens: %w", err)
	}

	// Fire-and-forget audit event
	if err := s.eventPublisher.PublishEvent(ctx, publisher.NewAuditEvent(userID, "user.logged_out_all", "success")); err != nil {
		slog.Warn("failed to publish audit event", "operation", "user.logged_out_all", "error", err)
	}

	return nil
}
