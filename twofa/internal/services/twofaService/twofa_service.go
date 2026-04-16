// Package twofaService implements 2FA orchestration with Shamir secret sharing,
// TOTP verification, backup codes, and MPC node coordination.
package twofaService

import (
	"context"
	"time"

	"github.com/vbncursed/vkr/twofa/internal/domain"
)

//go:generate minimock -i Storage -o ./mocks/ -s _mock.go
//go:generate minimock -i SessionStorage -o ./mocks/ -s _mock.go
//go:generate minimock -i MPCClient -o ./mocks/ -s _mock.go

// Storage defines the interface for TwoFA persistent data access.
type Storage interface {
	CreateTwoFARecord(ctx context.Context, userID string) error
	GetTwoFARecord(ctx context.Context, userID string) (*domain.TwoFARecord, error)
	StoreBatchBackupCodes(ctx context.Context, userID string, codeHashes []string) error
	EnableTwoFA(ctx context.Context, userID string) error
	DeleteTwoFARecord(ctx context.Context, userID string) error
	DeleteBackupCodes(ctx context.Context, userID string) error
	GetUnusedBackupCodeHashes(ctx context.Context, userID string) ([]domain.BackupCodeRow, error)
	MarkBackupCodeUsed(ctx context.Context, codeID string) error
}

// SessionStorage defines the interface for session/cache operations.
type SessionStorage interface {
	IncrementRateLimit(ctx context.Context, key string, ttl time.Duration) (int64, error)
	GetRateLimit(ctx context.Context, key string) (int64, error)
	SetUsedOTPCounter(ctx context.Context, userID string, counter int64, ttl time.Duration) error
	GetUsedOTPCounter(ctx context.Context, userID string) (int64, error)
	DeleteKeys(ctx context.Context, keys ...string) error
}

// MPCClient defines the domain-level interface for MPC node operations.
// Transport-agnostic — the bootstrap layer provides a gRPC adapter.
type MPCClient interface {
	StoreShare(ctx context.Context, userID string, shareIndex int, shareData []byte) error
	RetrieveShare(ctx context.Context, userID string, shareIndex int) ([]byte, error)
	DeleteShare(ctx context.Context, userID string) error
}

// TwoFAService implements 2FA orchestration business logic.
type TwoFAService struct {
	storage        Storage
	sessionStorage SessionStorage
	mpcClients     []MPCClient
	eventProducer  EventProducer
	mpcTimeout     time.Duration
}

// NewTwoFAService creates a new TwoFAService instance.
func NewTwoFAService(
	storage Storage,
	sessionStorage SessionStorage,
	mpcClients []MPCClient,
	eventProducer EventProducer,
	mpcTimeout time.Duration,
) *TwoFAService {
	return &TwoFAService{
		storage:        storage,
		sessionStorage: sessionStorage,
		mpcClients:     mpcClients,
		eventProducer:  eventProducer,
		mpcTimeout:     mpcTimeout,
	}
}
