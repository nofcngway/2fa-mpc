package mpcService

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/vbncursed/vkr/mpc/internal/storage/pgstorage"
)

// RetrieveShare retrieves and decrypts a share by user_id and share_index.
func (s *MPCService) RetrieveShare(ctx context.Context, userID string, shareIndex int) ([]byte, error) {
	share, err := s.storage.GetShare(ctx, userID, shareIndex)
	if err != nil {
		if errors.Is(err, pgstorage.ErrShareNotFound) {
			return nil, pgstorage.ErrShareNotFound
		}
		return nil, fmt.Errorf("get share: %w", err)
	}

	plaintext, err := s.decrypt(share.EncryptedData, share.Nonce)
	if err != nil {
		slog.Error("share decryption failed",
			"user_id", userID,
			"share_index", shareIndex,
			"node_id", s.nodeID,
		)
		return nil, fmt.Errorf("decrypt share: %w", err)
	}

	return plaintext, nil
}
