package authService

import (
	"context"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/google/uuid"

	"github.com/vbncursed/vkr/auth/internal/domain"
)

// Login authenticates a user by email and password, returning JWT tokens on success.
func (s *AuthService) Login(ctx context.Context, email, password string) (*domain.User, string, string, error) {
	// 1. Normalize email
	email = strings.ToLower(strings.TrimSpace(email))

	// 2. Look up user by email
	user, err := s.storage.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, "", "", err
	}
	if user == nil {
		return nil, "", "", domain.ErrInvalidCredentials
	}

	// 3. Verify password with bcrypt (same error for wrong password -- no credential enumeration)
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", "", domain.ErrInvalidCredentials
	}

	// 4. Generate new token family
	tokenFamily := uuid.New().String()

	// 5. Generate access token
	accessToken, _, err := s.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, "", "", err
	}

	// 6. Generate refresh token
	refreshToken, refreshJTI, err := s.GenerateRefreshToken(user.ID, user.Email, tokenFamily)
	if err != nil {
		return nil, "", "", err
	}

	// 7. Store refresh token in Redis
	if err := s.sessionStorage.StoreRefreshToken(ctx, refreshJTI, user.ID, tokenFamily, s.refreshTokenTTL); err != nil {
		return nil, "", "", err
	}

	return user, accessToken, refreshToken, nil
}
