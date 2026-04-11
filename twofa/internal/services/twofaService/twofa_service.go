package twofaService

import (
	"github.com/vbncursed/vkr/twofa/internal/storage/pgstorage"
	"github.com/vbncursed/vkr/twofa/internal/storage/redisstorage"
)

// Storage defines the interface for TwoFA persistent data access.
type Storage interface {
	// Methods added in Phase 7
}

// TwoFAService implements 2FA orchestration business logic.
type TwoFAService struct {
	storage        *pgstorage.PGStorage
	sessionStorage *redisstorage.RedisStorage
}

// NewTwoFAService creates a new TwoFAService instance.
func NewTwoFAService(storage *pgstorage.PGStorage, sessionStorage *redisstorage.RedisStorage) *TwoFAService {
	return &TwoFAService{
		storage:        storage,
		sessionStorage: sessionStorage,
	}
}
