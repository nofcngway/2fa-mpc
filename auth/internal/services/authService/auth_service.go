package authService

import (
	"context"
	"errors"

	"github.com/vbncursed/vkr/auth/internal/models"
	"github.com/vbncursed/vkr/auth/internal/storage/redisstorage"
)

// ErrDuplicateEmail indicates a user with this email already exists.
var ErrDuplicateEmail = errors.New("user with this email already exists")

// ErrInvalidEmail indicates the email format is invalid.
var ErrInvalidEmail = errors.New("invalid email format")

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
	sessionStorage *redisstorage.RedisStorage
}

// NewAuthService creates a new AuthService instance.
func NewAuthService(storage Storage, sessionStorage *redisstorage.RedisStorage) *AuthService {
	return &AuthService{
		storage:        storage,
		sessionStorage: sessionStorage,
	}
}
