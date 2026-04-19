// Package mpc_service_api provides gRPC handlers for the MPC Node service.
package mpc_service_api

import (
	"context"

	pb "github.com/vbncursed/vkr/mpc/internal/pb/mpc_api"
)

// mpcService defines the contract the API layer requires from the MPC service.
type mpcService interface {
	StoreShare(ctx context.Context, userID string, shareIndex int, shareData []byte) (string, error)
	RetrieveShare(ctx context.Context, userID string, shareIndex int) ([]byte, error)
	DeleteShare(ctx context.Context, userID string) (int64, error)
}

// MPCServiceAPI implements the gRPC MPCNodeServiceServer interface.
type MPCServiceAPI struct {
	pb.UnimplementedMPCNodeServiceServer
	service mpcService
}

// NewMPCServiceAPI creates a new MPCServiceAPI instance.
func NewMPCServiceAPI(service mpcService) *MPCServiceAPI {
	return &MPCServiceAPI{
		service: service,
	}
}
