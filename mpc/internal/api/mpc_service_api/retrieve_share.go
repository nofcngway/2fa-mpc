package mpc_service_api

import (
	"context"
	"errors"

	"github.com/vbncursed/vkr/mpc/internal/models"
	pb "github.com/vbncursed/vkr/mpc/internal/pb/mpc_api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RetrieveShare handles the RetrieveShare RPC call.
func (api *MPCServiceAPI) RetrieveShare(ctx context.Context, req *pb.RetrieveShareRequest) (*pb.RetrieveShareResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.GetShareIndex() < 0 {
		return nil, status.Error(codes.InvalidArgument, "share_index must be non-negative")
	}

	shareData, err := api.service.RetrieveShare(ctx, req.GetUserId(), int(req.GetShareIndex()))
	if err != nil {
		if errors.Is(err, models.ErrShareNotFound) {
			return nil, status.Error(codes.NotFound, "share not found")
		}
		return nil, status.Error(codes.Internal, "failed to retrieve share")
	}

	return &pb.RetrieveShareResponse{ShareData: shareData}, nil
}
