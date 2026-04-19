package bootstrap

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

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
	opts := grpc.WithTransportCredentials(insecure.NewCredentials())

	authConn, err := grpc.NewClient(cfg.AuthService.Addr, opts)
	if err != nil {
		return nil, fmt.Errorf("connect to auth service at %s: %w", cfg.AuthService.Addr, err)
	}

	twofaConn, err := grpc.NewClient(cfg.TwoFAService.Addr, opts)
	if err != nil {
		authConn.Close()
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
	if c.AuthConn != nil {
		c.AuthConn.Close()
	}
	if c.TwoFAConn != nil {
		c.TwoFAConn.Close()
	}
}
