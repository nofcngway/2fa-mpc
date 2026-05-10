package bootstrap

import (
	"fmt"

	"google.golang.org/grpc"

	"github.com/vbncursed/vkr/gateway/config"
	authpb "github.com/vbncursed/vkr/gateway/internal/pb/auth_api"
	twofapb "github.com/vbncursed/vkr/gateway/internal/pb/twofa_api"
)

type GRPCClients struct {
	AuthConn  *grpc.ClientConn
	TwoFAConn *grpc.ClientConn
	Auth      authpb.AuthServiceClient
	TwoFA     twofapb.TwoFAServiceClient
}

func NewGRPCClients(cfg *config.Config) (*GRPCClients, error) {
	transportCreds, err := clientTransportCreds(cfg)
	if err != nil {
		return nil, err
	}

	authConn, err := grpc.NewClient(cfg.AuthService.Addr, transportCreds)
	if err != nil {
		return nil, fmt.Errorf("connect to auth service at %s: %w", cfg.AuthService.Addr, err)
	}

	twofaConn, err := grpc.NewClient(cfg.TwoFAService.Addr, transportCreds)
	if err != nil {
		// Best-effort cleanup; close errors during failed dial are not actionable.
		_ = authConn.Close()
		return nil, fmt.Errorf("connect to twofa service at %s: %w", cfg.TwoFAService.Addr, err)
	}

	return &GRPCClients{
		AuthConn:  authConn,
		TwoFAConn: twofaConn,
		Auth:      authpb.NewAuthServiceClient(authConn),
		TwoFA:     twofapb.NewTwoFAServiceClient(twofaConn),
	}, nil
}

func (c *GRPCClients) Close() {
	// Close errors on shutdown are logged downstream; nothing to do for the
	// caller here, so the returns are deliberately ignored.
	if c.AuthConn != nil {
		_ = c.AuthConn.Close()
	}
	if c.TwoFAConn != nil {
		_ = c.TwoFAConn.Close()
	}
}
