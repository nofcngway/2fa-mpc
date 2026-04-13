package authService

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/google/uuid"

	"github.com/vbncursed/vkr/auth/internal/domain"
)

// COST_BCRYPT is the bcrypt hashing cost factor.
const COST_BCRYPT = 12

// Register creates a new user account with the given email and password.
// Returns the created user and JWT tokens for auto-login.
func (s *AuthService) Register(ctx context.Context, email, password string) (*domain.User, string, string, error) {
	// 1. Normalize email once upfront — all subsequent code uses the normalized form (IN-02)
	email = strings.ToLower(strings.TrimSpace(email))

	// 2. Validate email (basic format check on normalized form)
	if err := validateEmail(email); err != nil {
		return nil, "", "", err
	}

	// 3. Validate password (calls ValidatePassword from password_validation.go)
	if err := ValidatePassword(password); err != nil {
		return nil, "", "", err
	}

	// 4. Check if email already exists
	_, err := s.storage.GetUserByEmail(ctx, email)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, "", "", fmt.Errorf("check existing user: %w", err)
	}
	if err == nil {
		return nil, "", "", domain.ErrDuplicateEmail
	}

	// 5. Hash password with bcrypt cost=12
	hash, err := bcrypt.GenerateFromPassword([]byte(password), COST_BCRYPT)
	if err != nil {
		return nil, "", "", fmt.Errorf("hash password: %w", err)
	}

	// 6. Create user
	now := time.Now()
	user := &domain.User{
		ID:           uuid.New().String(),
		Email:        email,
		PasswordHash: string(hash),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.storage.CreateUser(ctx, user); err != nil {
		// Handle race condition: concurrent insert hit unique constraint
		if errors.Is(err, domain.ErrDuplicateEmail) {
			return nil, "", "", domain.ErrDuplicateEmail
		}
		return nil, "", "", fmt.Errorf("create user: %w", err)
	}

	// 7. Generate tokens for auto-login
	tokenFamily := uuid.New().String()

	accessToken, _, err := s.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, "", "", fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, refreshJTI, err := s.GenerateRefreshToken(user.ID, user.Email, tokenFamily)
	if err != nil {
		return nil, "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	if err := s.sessionStorage.StoreRefreshToken(ctx, refreshJTI, user.ID, tokenFamily, s.refreshTokenTTL); err != nil {
		return nil, "", "", fmt.Errorf("store refresh token: %w", err)
	}

	// Fire-and-forget audit event
	if err := s.eventProducer.PublishEvent(ctx, NewAuditEvent(user.ID, "user.registered", "success")); err != nil {
		slog.Warn("failed to publish audit event", "operation", "user.registered", "error", err)
	}

	return user, accessToken, refreshToken, nil
}

// validateEmail performs basic email format validation.
func validateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return domain.ErrInvalidEmail
	}
	if strings.Contains(email, " ") {
		return domain.ErrInvalidEmail
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return domain.ErrInvalidEmail
	}
	if parts[0] == "" || parts[1] == "" {
		return domain.ErrInvalidEmail
	}
	if !strings.Contains(parts[1], ".") {
		return domain.ErrInvalidEmail
	}
	return nil
}
