package authService

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/google/uuid"

	"github.com/vbncursed/vkr/auth/internal/domain"
)

// dummyHash is a pre-computed bcrypt hash used for timing-safe comparison when user is not found.
// This prevents user enumeration via response time differences.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-safe-dummy"), COST_BCRYPT)

// Login authenticates a user by email and password, returning JWT tokens on success.
func (s *AuthService) Login(ctx context.Context, email, password string) (*domain.User, string, string, error) {
	// 1. Normalize email
	email = strings.ToLower(strings.TrimSpace(email))

	// 2. Look up user by email
	user, err := s.storage.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			// Dummy bcrypt compare to prevent timing oracle (constant response time)
			_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))

			// Audit failed attempt (fire-and-forget)
			if err := s.eventProducer.PublishEvent(ctx, NewAuditEvent(email, "user.login_failed", "failure")); err != nil {
				slog.Warn("failed to publish audit event", "operation", "user.login_failed", "error", err)
			}
			return nil, "", "", domain.ErrInvalidCredentials
		}
		return nil, "", "", fmt.Errorf("get user by email: %w", err)
	}

	// 3. Verify password with bcrypt (same error for wrong password -- no credential enumeration)
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		// Audit failed attempt (fire-and-forget)
		if err := s.eventProducer.PublishEvent(ctx, NewAuditEvent(user.ID, "user.login_failed", "failure")); err != nil {
			slog.Warn("failed to publish audit event", "operation", "user.login_failed", "error", err)
		}
		return nil, "", "", domain.ErrInvalidCredentials
	}

	// 4. Generate new token family
	tokenFamily := uuid.New().String()

	// 5. Generate access token
	accessToken, _, err := s.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, "", "", fmt.Errorf("generate access token: %w", err)
	}

	// 6. Generate refresh token
	refreshToken, refreshJTI, err := s.GenerateRefreshToken(user.ID, user.Email, tokenFamily)
	if err != nil {
		return nil, "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	// 7. Store refresh token in Redis
	if err := s.sessionStorage.StoreRefreshToken(ctx, refreshJTI, user.ID, tokenFamily, s.refreshTokenTTL); err != nil {
		return nil, "", "", fmt.Errorf("store refresh token: %w", err)
	}

	// Fire-and-forget audit event
	if err := s.eventProducer.PublishEvent(ctx, NewAuditEvent(user.ID, "user.logged_in", "success")); err != nil {
		slog.Warn("failed to publish audit event", "operation", "user.logged_in", "error", err)
	}

	return user, accessToken, refreshToken, nil
}
