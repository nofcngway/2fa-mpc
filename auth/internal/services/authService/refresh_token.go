package authService

import (
	"context"

	"github.com/vbncursed/vkr/auth/internal/domain"
)

// RefreshToken performs token rotation with theft detection.
// If the JTI exists in Redis, it rotates (delete old, issue new with same family).
// If the JTI is missing but the JWT is valid, it signals token theft and revokes the entire family.
func (s *AuthService) RefreshToken(ctx context.Context, refreshTokenStr string) (string, string, error) {
	// 1. Parse and validate the refresh JWT
	claims, err := s.ParseToken(refreshTokenStr)
	if err != nil {
		return "", "", domain.ErrInvalidToken
	}

	// 2. Look up JTI in Redis
	tokenData, err := s.sessionStorage.GetRefreshToken(ctx, claims.ID)
	if err != nil {
		return "", "", err
	}

	// 3. Theft detection: valid JWT but JTI not in Redis
	if tokenData == nil {
		// Token was already rotated -- this is a reuse attempt (stolen token)
		_ = s.sessionStorage.DeleteTokenFamily(ctx, claims.TokenFamily)
		return "", "", domain.ErrTokenRevoked
	}

	// 4. Normal rotation: delete old JTI
	if err := s.sessionStorage.DeleteRefreshToken(ctx, claims.ID); err != nil {
		return "", "", err
	}

	// 5. Generate new access token
	newAccess, _, err := s.GenerateAccessToken(tokenData.UserID, claims.Email)
	if err != nil {
		return "", "", err
	}

	// 6. Generate new refresh token with same family
	newRefresh, newJTI, err := s.GenerateRefreshToken(tokenData.UserID, claims.Email, tokenData.TokenFamily)
	if err != nil {
		return "", "", err
	}

	// 7. Store new refresh token
	if err := s.sessionStorage.StoreRefreshToken(ctx, newJTI, tokenData.UserID, tokenData.TokenFamily, s.refreshTokenTTL); err != nil {
		return "", "", err
	}

	return newAccess, newRefresh, nil
}
