package bootstrap

import (
	"fmt"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/vbncursed/vkr/twofa/config"
)

// mpcTransportCreds picks the transport credential dial option for outbound
// MPC calls. When cfg.TLS.Enabled is true the function loads mTLS credentials;
// otherwise it returns plaintext credentials with a loud warning. Production
// deployments must enable TLS; the insecure fallback exists only for local
// development.
func mpcTransportCreds(cfg *config.Config) (grpc.DialOption, error) {
	if cfg.TLS.Enabled {
		creds, err := loadClientTLSCredentials(cfg.TLS.CertFile, cfg.TLS.KeyFile, cfg.TLS.CAFile)
		if err != nil {
			return nil, fmt.Errorf("load mtls credentials for mpc client: %w", err)
		}
		slog.Info("mTLS enabled for MPC clients", "cert", cfg.TLS.CertFile)
		return grpc.WithTransportCredentials(creds), nil
	}
	slog.Warn("MPC clients running insecure — set tls.enabled=true for production")
	return grpc.WithTransportCredentials(insecure.NewCredentials()), nil
}
