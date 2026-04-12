package mpc_service_api

import (
	"context"

	pb "github.com/vbncursed/vkr/mpc/internal/pb/mpc_api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DeleteShare handles the DeleteShare RPC call.
func (api *MPCServiceAPI) DeleteShare(ctx context.Context, req *pb.DeleteShareRequest) (*pb.DeleteShareResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	count, err := api.service.DeleteShare(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to delete shares")
	}

	return &pb.DeleteShareResponse{DeletedCount: int32(count)}, nil
}
