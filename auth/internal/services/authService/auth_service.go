package authService

import (
	"github.com/vbncursed/vkr/auth/internal/storage/pgstorage"
	"github.com/vbncursed/vkr/auth/internal/storage/redisstorage"
)

// Storage defines the interface for persistent data access.
type Storage interface {
	// Methods added in Phase 2 (CreateUser, GetUserByEmail, etc.)
}

// SessionStorage defines the interface for session/cache operations.
type SessionStorage interface {
	// Methods added in Phase 3 (StoreRefreshToken, DeleteRefreshToken, etc.)
}

// AuthService implements authentication business logic.
type AuthService struct {
	storage        *pgstorage.PGStorage
	sessionStorage *redisstorage.RedisStorage
}

// NewAuthService creates a new AuthService instance.
func NewAuthService(storage *pgstorage.PGStorage, sessionStorage *redisstorage.RedisStorage) *AuthService {
	return &AuthService{
		storage:        storage,
		sessionStorage: sessionStorage,
	}
}
