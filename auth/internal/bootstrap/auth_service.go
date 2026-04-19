package bootstrap

import (
	"fmt"

	"github.com/vbncursed/vkr/auth/config"
	"github.com/vbncursed/vkr/auth/internal/services/authService"
)

// NewAuthService creates a new AuthService with the provided storage dependencies and RSA keys.
func NewAuthService(
	cfg *config.Config,
	storage authService.Storage,
	sessionStorage authService.SessionStorage,
	eventProducer authService.EventProducer,
) (*authService.AuthService, error) {
	privateKey, publicKey, err := authService.LoadRSAKeys(cfg.JWT.PrivateKeyPath, cfg.JWT.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load RSA keys: %w", err)
	}

	return authService.NewAuthService(authService.Deps{
		Storage:         storage,
		SessionStorage:  sessionStorage,
		EventProducer:   eventProducer,
		PrivateKey:      privateKey,
		PublicKey:        publicKey,
		AccessTokenTTL:  cfg.JWT.AccessTokenTTL,
		RefreshTokenTTL: cfg.JWT.RefreshTokenTTL,
	})
}
