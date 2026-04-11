// Package mpcService implements MPC node business logic.
package mpcService

import "github.com/vbncursed/vkr/mpc/internal/storage/pgstorage"

// Storage defines the interface for share persistent data access.
type Storage interface {
	// Methods added in Phase 6
}

// MPCService implements MPC node business logic.
type MPCService struct {
	storage       *pgstorage.PGStorage
	encryptionKey []byte
	nodeID        int
}

// NewMPCService creates a new MPCService instance.
func NewMPCService(storage *pgstorage.PGStorage, encryptionKey []byte, nodeID int) *MPCService {
	return &MPCService{
		storage:       storage,
		encryptionKey: encryptionKey,
		nodeID:        nodeID,
	}
}
