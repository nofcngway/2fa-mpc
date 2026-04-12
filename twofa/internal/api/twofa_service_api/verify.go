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

// Verify2FA validates a TOTP code for a user.
func (api *TwoFAServiceAPI) Verify2FA(ctx context.Context, req *pb.Verify2FARequest) (*pb.Verify2FAResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.GetOtpCode() == "" {
		return nil, status.Error(codes.InvalidArgument, "otp_code is required")
	}

	valid, isNewlyEnabled, err := api.service.Verify(ctx, req.GetUserId(), req.GetOtpCode())
	if err != nil {
		switch {
		case errors.Is(err, twofaService.ErrRateLimitExceeded):
			return nil, status.Error(codes.ResourceExhausted, "too many verification attempts")
		case errors.Is(err, twofaService.ErrOTPReused):
			return nil, status.Error(codes.InvalidArgument, "OTP code already used")
		case errors.Is(err, twofaService.ErrNotSetUp):
			return nil, status.Error(codes.FailedPrecondition, "2FA not set up")
		default:
			slog.Error("verify 2fa failed", "user_id", req.GetUserId(), "error", err)
			return nil, status.Error(codes.Internal, "verification failed")
		}
	}

	return &pb.Verify2FAResponse{
		Valid:          valid,
		IsNewlyEnabled: isNewlyEnabled,
	}, nil
}
