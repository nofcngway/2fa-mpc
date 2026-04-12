package authService

import (
	"context"

	"github.com/vbncursed/vkr/auth/internal/domain"
)

// Logout invalidates a single session by deleting its refresh token from Redis.
func (s *AuthService) Logout(ctx context.Context, refreshTokenStr string) error {
	claims, err := s.ParseToken(refreshTokenStr)
	if err != nil {
		return domain.ErrInvalidToken
	}

	return s.sessionStorage.DeleteRefreshToken(ctx, claims.ID)
}
