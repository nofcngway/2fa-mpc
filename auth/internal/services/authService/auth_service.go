package authService

import (
	"context"

	"github.com/vbncursed/vkr/auth/internal/models"
)

// Storage defines the interface for persistent data access.
type Storage interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
}

// SessionStorage defines the interface for session/cache operations.
type SessionStorage interface {
	// Methods added in Phase 3 (StoreRefreshToken, DeleteRefreshToken, etc.)
}

// AuthService implements authentication business logic.
type AuthService struct {
	storage        Storage
	sessionStorage SessionStorage
}

// NewAuthService creates a new AuthService instance.
func NewAuthService(storage Storage, sessionStorage SessionStorage) *AuthService {
	return &AuthService{
		storage:        storage,
		sessionStorage: sessionStorage,
	}
}
