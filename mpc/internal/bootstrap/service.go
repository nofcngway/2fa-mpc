package bootstrap

import (
	"fmt"

	"github.com/vbncursed/vkr/mpc/config"
	"github.com/vbncursed/vkr/mpc/internal/services/mpcService"
)

// NewMPCService creates a new MPC business logic service.
// Returns error if encryption key is not exactly 32 bytes.
func NewMPCService(storage mpcService.Storage, cfg *config.Config, eventProducer mpcService.EventProducer) (*mpcService.MPCService, error) {
	key := []byte(cfg.Node.EncryptionKey)
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be exactly 32 bytes, got %d", len(key))
	}
	return mpcService.NewMPCService(
		storage,
		key,
		cfg.Node.ID,
		eventProducer,
	), nil
}
