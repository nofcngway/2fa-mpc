package mpc_service_api

import (
	"context"

	pb "github.com/vbncursed/vkr/mpc/internal/pb/mpc_api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StoreShare handles the StoreShare RPC call.
func (api *MPCServiceAPI) StoreShare(ctx context.Context, req *pb.StoreShareRequest) (*pb.StoreShareResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}
