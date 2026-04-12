package authService

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/google/uuid"

	"github.com/vbncursed/vkr/auth/internal/domain"
	"github.com/vbncursed/vkr/auth/internal/models"
)

// COST_BCRYPT is the bcrypt hashing cost factor.
const COST_BCRYPT = 12

// Register creates a new user account with the given email and password.
func (s *AuthService) Register(ctx context.Context, email, password string) (*models.User, error) {
	// 1. Validate email (basic format check)
	if err := validateEmail(email); err != nil {
		return nil, err
	}

	// 2. Validate password (calls ValidatePassword from password_validation.go)
	if err := ValidatePassword(password); err != nil {
		return nil, err
	}

	// 3. Check if email already exists
	existing, err := s.storage.GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return nil, fmt.Errorf("check existing user: %w", err)
	}
	if existing != nil {
		return nil, domain.ErrDuplicateEmail
	}

	// 4. Hash password with bcrypt cost=12
	hash, err := bcrypt.GenerateFromPassword([]byte(password), COST_BCRYPT)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// 5. Create user
	now := time.Now()
	user := &models.User{
		ID:           uuid.New().String(),
		Email:        strings.ToLower(strings.TrimSpace(email)),
		PasswordHash: string(hash),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.storage.CreateUser(ctx, user); err != nil {
		// Handle race condition: concurrent insert hit unique constraint
		if errors.Is(err, domain.ErrDuplicateEmail) {
			return nil, domain.ErrDuplicateEmail
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
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
