package twofaService

import (
	"context"
	"time"

	"google.golang.org/grpc"

	"github.com/vbncursed/vkr/twofa/internal/pb/mpc_api"
	"github.com/vbncursed/vkr/twofa/internal/models"
)

//go:generate minimock -i Storage -o ./mocks/ -s _mock.go
//go:generate minimock -i SessionStorage -o ./mocks/ -s _mock.go
//go:generate minimock -i MPCClient -o ./mocks/ -s _mock.go

// Storage defines the interface for TwoFA persistent data access.
type Storage interface {
	CreateTwoFARecord(ctx context.Context, userID string) error
	GetTwoFARecord(ctx context.Context, userID string) (*models.TwoFARecord, error)
	StoreBatchBackupCodes(ctx context.Context, userID string, codeHashes []string) error
}

// SessionStorage defines the interface for session/cache operations.
type SessionStorage interface {
	// Methods added in Phase 8 (rate limiting)
}

// MPCClient defines the interface for MPC node gRPC operations.
// Mirrors mpc_api.MPCNodeServiceClient for testability via minimock.
type MPCClient interface {
	StoreShare(ctx context.Context, in *mpc_api.StoreShareRequest, opts ...grpc.CallOption) (*mpc_api.StoreShareResponse, error)
	RetrieveShare(ctx context.Context, in *mpc_api.RetrieveShareRequest, opts ...grpc.CallOption) (*mpc_api.RetrieveShareResponse, error)
	DeleteShare(ctx context.Context, in *mpc_api.DeleteShareRequest, opts ...grpc.CallOption) (*mpc_api.DeleteShareResponse, error)
}

// TwoFAService implements 2FA orchestration business logic.
type TwoFAService struct {
	storage        Storage
	sessionStorage SessionStorage
	mpcClients     []MPCClient
	sharedSecret   string
	mpcTimeout     time.Duration
}

// NewTwoFAService creates a new TwoFAService instance.
func NewTwoFAService(
	storage Storage,
	sessionStorage SessionStorage,
	mpcClients []MPCClient,
	sharedSecret string,
	mpcTimeout time.Duration,
) *TwoFAService {
	return &TwoFAService{
		storage:        storage,
		sessionStorage: sessionStorage,
		mpcClients:     mpcClients,
		sharedSecret:   sharedSecret,
		mpcTimeout:     mpcTimeout,
	}
}
