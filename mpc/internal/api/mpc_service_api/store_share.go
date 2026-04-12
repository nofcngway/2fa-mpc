package mpc_service_api

import (
	"context"
	"errors"

	pb "github.com/vbncursed/vkr/mpc/internal/pb/mpc_api"
	"github.com/vbncursed/vkr/mpc/internal/storage/pgstorage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StoreShare handles the StoreShare RPC call.
func (api *MPCServiceAPI) StoreShare(ctx context.Context, req *pb.StoreShareRequest) (*pb.StoreShareResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if len(req.GetShareData()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "share_data is required")
	}
	if req.GetShareIndex() < 0 {
		return nil, status.Error(codes.InvalidArgument, "share_index must be non-negative")
	}

	shareID, err := api.service.StoreShare(ctx, req.GetUserId(), int(req.GetShareIndex()), req.GetShareData())
	if err != nil {
		if errors.Is(err, pgstorage.ErrDuplicateShare) {
			return nil, status.Error(codes.AlreadyExists, "share already exists for this user and index")
		}
		return nil, status.Error(codes.Internal, "failed to store share")
	}

	return &pb.StoreShareResponse{ShareId: shareID}, nil
}
