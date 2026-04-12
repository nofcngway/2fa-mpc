// Package mpcService implements MPC node business logic.
package mpcService

import (
	"context"

	"github.com/vbncursed/vkr/mpc/internal/models"
)

//go:generate minimock -i Storage -o ./mocks -g -s _mock.go

// Storage defines the interface for share persistent data access.
type Storage interface {
	CreateShare(ctx context.Context, share *models.Share) error
	GetShare(ctx context.Context, userID string, shareIndex int) (*models.Share, error)
	DeleteSharesByUserID(ctx context.Context, userID string) (int64, error)
}

// MPCService implements MPC node business logic.
type MPCService struct {
	storage       Storage
	encryptionKey []byte
	nodeID        int
}

// NewMPCService creates a new MPCService instance.
func NewMPCService(storage Storage, encryptionKey []byte, nodeID int) *MPCService {
	return &MPCService{
		storage:       storage,
		encryptionKey: encryptionKey,
		nodeID:        nodeID,
	}
}
