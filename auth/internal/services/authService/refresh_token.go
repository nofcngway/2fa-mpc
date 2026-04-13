package authService

import (
	"context"
	"log/slog"

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
	if claims.TokenType != "refresh" {
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
		if err := s.sessionStorage.DeleteTokenFamily(ctx, claims.TokenFamily, claims.Subject); err != nil {
			slog.Error("failed to revoke token family after theft detection", "family", claims.TokenFamily, "user_id", claims.Subject, "error", err)
		}

		// Fire-and-forget audit event for reuse detection
		if err := s.eventProducer.PublishEvent(ctx, NewAuditEvent(claims.Subject, "token.refresh_reuse_detected", "alert")); err != nil {
			slog.Warn("failed to publish audit event", "operation", "token.refresh_reuse_detected", "error", err)
		}

		return "", "", domain.ErrTokenRevoked
	}

	// 4. Generate new access token
	newAccess, _, err := s.GenerateAccessToken(tokenData.UserID, claims.Email)
	if err != nil {
		return "", "", err
	}

	// 5. Generate new refresh token with same family
	newRefresh, newJTI, err := s.GenerateRefreshToken(tokenData.UserID, claims.Email, tokenData.TokenFamily)
	if err != nil {
		return "", "", err
	}

	// 6. Store new token BEFORE deleting old — safer failure mode:
	// if store fails, old token remains valid and user can retry;
	// if delete fails after store, old token expires naturally at TTL.
	if err := s.sessionStorage.StoreRefreshToken(ctx, newJTI, tokenData.UserID, tokenData.TokenFamily, s.refreshTokenTTL); err != nil {
		return "", "", err
	}

	// 7. Delete old JTI (best-effort — it will expire naturally on TTL if this fails)
	_ = s.sessionStorage.DeleteRefreshToken(ctx, claims.ID)

	// Fire-and-forget audit event
	if err := s.eventProducer.PublishEvent(ctx, NewAuditEvent(tokenData.UserID, "token.refreshed", "success")); err != nil {
		slog.Warn("failed to publish audit event", "operation", "token.refreshed", "error", err)
	}

	return newAccess, newRefresh, nil
}
