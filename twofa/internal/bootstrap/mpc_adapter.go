package bootstrap

import (
	"context"
	"fmt"

	"github.com/vbncursed/vkr/twofa/internal/pb/mpc_api"
	"google.golang.org/grpc"
)

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
