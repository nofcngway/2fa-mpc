package bootstrap

import (
	"fmt"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/vbncursed/vkr/gateway/config"
)

// clientTransportCreds picks the gRPC dial option for outbound calls from the
// Gateway to Auth and TwoFA. When cfg.TLS.Enabled is true the function loads
// mTLS credentials; otherwise it falls back to plaintext with a loud warning
// (intended only for local development). Production must always enable TLS.
func clientTransportCreds(cfg *config.Config) (grpc.DialOption, error) {
	if cfg.TLS.Enabled {
		creds, err := loadClientTLSCredentials(cfg.TLS.CertFile, cfg.TLS.KeyFile, cfg.TLS.CAFile)
		if err != nil {
			return nil, fmt.Errorf("load mtls credentials for gateway client: %w", err)
		}
		slog.Info("mTLS enabled for Gateway clients", "cert", cfg.TLS.CertFile)
		return grpc.WithTransportCredentials(creds), nil
	}
	slog.Warn("Gateway running insecure — set tls.enabled=true for production")
	return grpc.WithTransportCredentials(insecure.NewCredentials()), nil
}
