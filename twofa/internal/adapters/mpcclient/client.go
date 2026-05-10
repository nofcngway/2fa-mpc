// Package mpcclient adapts the generated gRPC MPCNodeServiceClient to the
// domain-level twofaService.MPCClient interface (a port in hexagonal terms).
//
// The package is the only place in the TwoFA service that imports the
// generated protobuf types for the MPC API; the rest of the code talks to
// the domain interface only. This keeps gRPC concerns out of the use-case
// layer.
package mpcclient

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	"github.com/vbncursed/vkr/twofa/internal/pb/mpc_api"
	"github.com/vbncursed/vkr/twofa/internal/services/twofaService"
)

// adapter implements twofaService.MPCClient over a gRPC connection.
type adapter struct {
	client mpc_api.MPCNodeServiceClient
}

// New wraps a gRPC connection and returns it as a domain MPCClient.
func New(conn grpc.ClientConnInterface) twofaService.MPCClient {
	return &adapter{client: mpc_api.NewMPCNodeServiceClient(conn)}
}

func (a *adapter) StoreShare(ctx context.Context, userID string, shareIndex int, shareData []byte) error {
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

func (a *adapter) RetrieveShare(ctx context.Context, userID string, shareIndex int) ([]byte, error) {
	resp, err := a.client.RetrieveShare(ctx, &mpc_api.RetrieveShareRequest{
		UserId:     userID,
		ShareIndex: int32(shareIndex),
	})
	if err != nil {
		return nil, fmt.Errorf("grpc RetrieveShare: %w", err)
	}
	return resp.ShareData, nil
}

func (a *adapter) DeleteShare(ctx context.Context, userID string) error {
	_, err := a.client.DeleteShare(ctx, &mpc_api.DeleteShareRequest{
		UserId: userID,
	})
	if err != nil {
		return fmt.Errorf("grpc DeleteShare: %w", err)
	}
	return nil
}
