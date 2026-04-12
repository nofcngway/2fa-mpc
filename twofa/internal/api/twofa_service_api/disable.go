package twofa_service_api

import (
	"context"
	"errors"
	"log/slog"

	pb "github.com/vbncursed/vkr/twofa/internal/pb/twofa_api"
	"github.com/vbncursed/vkr/twofa/internal/services/twofaService"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Disable2FA removes 2FA enrollment for a user after OTP verification.
func (api *TwoFAServiceAPI) Disable2FA(ctx context.Context, req *pb.Disable2FARequest) (*pb.Disable2FAResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.GetOtpCode() == "" {
		return nil, status.Error(codes.InvalidArgument, "otp_code is required")
	}

	err := api.service.Disable(ctx, req.GetUserId(), req.GetOtpCode())
	if err != nil {
		switch {
		case errors.Is(err, twofaService.ErrNotSetUp):
			return nil, status.Error(codes.FailedPrecondition, "2FA not set up")
		case errors.Is(err, twofaService.ErrNotEnabled):
			return nil, status.Error(codes.FailedPrecondition, "2FA not enabled")
		default:
			slog.Error("disable 2fa failed", "user_id", req.GetUserId(), "error", err)
			return nil, status.Error(codes.Internal, "disable failed")
		}
	}

	return &pb.Disable2FAResponse{}, nil
}
