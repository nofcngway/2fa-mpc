package authService

import (
	"context"
	"log/slog"

	"github.com/vbncursed/vkr/auth/internal/domain"
)

// Logout invalidates a single session by deleting its refresh token from Redis.
func (s *AuthService) Logout(ctx context.Context, refreshTokenStr string) error {
	claims, err := s.ParseToken(refreshTokenStr)
	if err != nil {
		return domain.ErrInvalidToken
	}

	if err := s.sessionStorage.DeleteRefreshToken(ctx, claims.ID); err != nil {
		return err
	}

	// Fire-and-forget audit event
	if err := s.eventProducer.PublishEvent(ctx, NewAuditEvent(claims.Subject, "user.logged_out", "success")); err != nil {
		slog.Warn("failed to publish audit event", "operation", "user.logged_out", "error", err)
	}

	return nil
}
