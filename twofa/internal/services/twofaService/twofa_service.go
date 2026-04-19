// Package twofaService implements 2FA orchestration with Shamir secret sharing,
// TOTP verification, backup codes, and MPC node coordination.
package twofaService

import (
	"context"
	"errors"
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

// Deps groups all dependencies required by TwoFAService.
type Deps struct {
	Storage        Storage
	SessionStorage SessionStorage
	MPCClients     []MPCClient
	EventProducer  EventProducer
	MPCTimeout     time.Duration
}

// NewTwoFAService creates a new TwoFAService instance. Returns an error if any required dependency is nil.
func NewTwoFAService(d Deps) (*TwoFAService, error) {
	var errs []error
	if d.Storage == nil {
		errs = append(errs, errors.New("storage is required"))
	}
	if d.SessionStorage == nil {
		errs = append(errs, errors.New("session storage is required"))
	}
	if len(d.MPCClients) == 0 {
		errs = append(errs, errors.New("mpc clients are required"))
	}
	if d.EventProducer == nil {
		errs = append(errs, errors.New("event producer is required"))
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return &TwoFAService{
		storage:        d.Storage,
		sessionStorage: d.SessionStorage,
		mpcClients:     d.MPCClients,
		eventProducer:  d.EventProducer,
		mpcTimeout:     d.MPCTimeout,
	}, nil
}
