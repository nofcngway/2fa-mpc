package mpc_service_api

import (
	"context"

	pb "github.com/vbncursed/vkr/mpc/internal/pb/mpc_api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DeleteShare handles the DeleteShare RPC call.
func (api *MPCServiceAPI) DeleteShare(ctx context.Context, req *pb.DeleteShareRequest) (*pb.DeleteShareResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}
