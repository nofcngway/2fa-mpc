package mpcService

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/vbncursed/vkr/mpc/internal/models"
	"github.com/vbncursed/vkr/mpc/internal/storage/pgstorage"
)

// StoreShare encrypts share data and persists it.
// Returns the generated share ID.
func (s *MPCService) StoreShare(ctx context.Context, userID string, shareIndex int, shareData []byte) (string, error) {
	encryptedData, nonce, err := s.encrypt(shareData)
	if err != nil {
		return "", fmt.Errorf("encrypt share: %w", err)
	}

	share := &models.Share{
		ID:            uuid.New().String(),
		UserID:        userID,
		ShareIndex:    shareIndex,
		EncryptedData: encryptedData,
		Nonce:         nonce,
		CreatedAt:     time.Now(),
	}

	if err := s.storage.CreateShare(ctx, share); err != nil {
		if errors.Is(err, pgstorage.ErrDuplicateShare) {
			return "", pgstorage.ErrDuplicateShare
		}
		return "", fmt.Errorf("store share: %w", err)
	}

	return share.ID, nil
}
