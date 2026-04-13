package bootstrap

import (
	"github.com/vbncursed/vkr/twofa/config"
	"github.com/vbncursed/vkr/twofa/internal/services/twofaService"
)

// NewTwoFAService creates a new TwoFA business logic service.
func NewTwoFAService(
	storage twofaService.Storage,
	sessionStorage twofaService.SessionStorage,
	mpcClients []twofaService.MPCClient,
	eventProducer twofaService.EventProducer,
	cfg *config.Config,
) *twofaService.TwoFAService {
	return twofaService.NewTwoFAService(
		storage,
		sessionStorage,
		mpcClients,
		eventProducer,
		cfg.SharedSecret,
		cfg.GetMPCTimeout(),
	)
}
