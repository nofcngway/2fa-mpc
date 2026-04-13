package authService

import (
	"context"

	"github.com/vbncursed/vkr/auth/internal/domain"
)

// ValidateToken parses an access token and returns the user_id and email from its claims.
func (s *AuthService) ValidateToken(_ context.Context, accessTokenStr string) (string, string, error) {
	claims, err := s.ParseToken(accessTokenStr)
	if err != nil {
		return "", "", err
	}

	if claims.Subject == "" || claims.TokenType != "access" {
		return "", "", domain.ErrInvalidToken
	}

	return claims.Subject, claims.Email, nil
}
