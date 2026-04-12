package twofa_service_api

import (
	"context"
	"log/slog"
	"time"

	pb "github.com/vbncursed/vkr/twofa/internal/pb/twofa_api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Get2FAStatus returns the current 2FA enrollment status for a user.
func (api *TwoFAServiceAPI) Get2FAStatus(ctx context.Context, req *pb.Get2FAStatusRequest) (*pb.Get2FAStatusResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	record, err := api.service.GetStatus(ctx, req.GetUserId())
	if err != nil {
		slog.Error("get 2fa status failed", "user_id", req.GetUserId(), "error", err)
		return nil, status.Error(codes.Internal, "failed to get status")
	}

	if record == nil {
		return &pb.Get2FAStatusResponse{
			IsEnabled: false,
			CreatedAt: "",
		}, nil
	}

	return &pb.Get2FAStatusResponse{
		IsEnabled: record.IsEnabled,
		CreatedAt: record.CreatedAt.Format(time.RFC3339),
	}, nil
}
