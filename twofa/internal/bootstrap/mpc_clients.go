package bootstrap

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/vbncursed/vkr/twofa/config"
	"github.com/vbncursed/vkr/twofa/internal/pb/mpc_api"
	"github.com/vbncursed/vkr/twofa/internal/services/twofaService"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// NewMPCClients creates gRPC connections to all MPC nodes from config.
// Returns MPCClient slice (satisfying twofaService.MPCClient interface) and
// a slice of io.Closer for graceful shutdown of connections.
func NewMPCClients(cfg *config.Config) ([]twofaService.MPCClient, []io.Closer, error) {
	clients := make([]twofaService.MPCClient, len(cfg.MPCNodes))
	conns := make([]io.Closer, len(cfg.MPCNodes))

	for i, node := range cfg.MPCNodes {
		var transportCreds grpc.DialOption
		if cfg.MPCInsecure {
			slog.Warn("using insecure MPC connection", "node", node.Addr)
			transportCreds = grpc.WithTransportCredentials(insecure.NewCredentials())
		} else {
			// TODO: configure TLS/mTLS certificates for production deployment
			transportCreds = grpc.WithTransportCredentials(insecure.NewCredentials())
		}

		conn, err := grpc.NewClient(node.Addr,
			transportCreds,
			grpc.WithUnaryInterceptor(authMetadataInterceptor(cfg.SharedSecret)),
		)
		if err != nil {
			for j := range i {
				conns[j].Close()
			}
			return nil, nil, fmt.Errorf("connect to MPC node %d at %s: %w", i, node.Addr, err)
		}
		clients[i] = newMPCClientAdapter(conn)
		conns[i] = conn
	}
	return clients, conns, nil
}

// authMetadataInterceptor returns a gRPC unary client interceptor that
// attaches the shared secret in "authorization" metadata on every outgoing call.
func authMetadataInterceptor(secret string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", secret)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// mpcClientAdapter wraps a generated gRPC MPCNodeServiceClient to implement
// the domain-level twofaService.MPCClient interface.
type mpcClientAdapter struct {
	client mpc_api.MPCNodeServiceClient
}

func (a *mpcClientAdapter) StoreShare(ctx context.Context, userID string, shareIndex int, shareData []byte) error {
	_, err := a.client.StoreShare(ctx, &mpc_api.StoreShareRequest{
		UserId:     userID,
		ShareIndex: int32(shareIndex),
		ShareData:  shareData,
	})
	if err != nil {
		return fmt.Errorf("grpc StoreShare: %w", err)
	}
	return nil
}

func (a *mpcClientAdapter) RetrieveShare(ctx context.Context, userID string, shareIndex int) ([]byte, error) {
	resp, err := a.client.RetrieveShare(ctx, &mpc_api.RetrieveShareRequest{
		UserId:     userID,
		ShareIndex: int32(shareIndex),
	})
	if err != nil {
		return nil, fmt.Errorf("grpc RetrieveShare: %w", err)
	}
	return resp.ShareData, nil
}

func (a *mpcClientAdapter) DeleteShare(ctx context.Context, userID string) error {
	_, err := a.client.DeleteShare(ctx, &mpc_api.DeleteShareRequest{
		UserId: userID,
	})
	if err != nil {
		return fmt.Errorf("grpc DeleteShare: %w", err)
	}
	return nil
}

// newMPCClientAdapter wraps a gRPC connection into a domain MPCClient.
func newMPCClientAdapter(conn grpc.ClientConnInterface) *mpcClientAdapter {
	return &mpcClientAdapter{client: mpc_api.NewMPCNodeServiceClient(conn)}
}
