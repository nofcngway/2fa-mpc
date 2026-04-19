// Package mpcService implements MPC node business logic.
package mpcService

import (
	"context"
	"errors"

	"github.com/vbncursed/vkr/mpc/internal/domain"
)

//go:generate minimock -i Storage -o ./mocks -g -s _mock.go

// Storage defines the interface for share persistent data access.
type Storage interface {
	CreateShare(ctx context.Context, share *domain.Share) error
	GetShare(ctx context.Context, userID string, shareIndex int) (*domain.Share, error)
	DeleteSharesByUserID(ctx context.Context, userID string) (int64, error)
}

// Deps groups all dependencies required by MPCService.
type Deps struct {
	Storage       Storage
	EncryptionKey []byte
	NodeID        int
	EventProducer EventProducer
}

// MPCService implements MPC node business logic.
type MPCService struct {
	storage       Storage
	encryptionKey []byte
	nodeID        int
	eventProducer EventProducer
}

// NewMPCService creates a new MPCService instance. Returns an error if any required dependency is nil.
func NewMPCService(d Deps) (*MPCService, error) {
	var errs []error
	if d.Storage == nil {
		errs = append(errs, errors.New("storage is required"))
	}
	if len(d.EncryptionKey) == 0 {
		errs = append(errs, errors.New("encryption key is required"))
	}
	if d.EventProducer == nil {
		errs = append(errs, errors.New("event producer is required"))
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return &MPCService{
		storage:       d.Storage,
		encryptionKey: d.EncryptionKey,
		nodeID:        d.NodeID,
		eventProducer: d.EventProducer,
	}, nil
}
