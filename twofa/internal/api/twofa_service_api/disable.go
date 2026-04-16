package twofa_service_api

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"github.com/vbncursed/vkr/twofa/internal/domain"
	pb "github.com/vbncursed/vkr/twofa/internal/pb/twofa_api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Disable2FA removes 2FA enrollment for a user after OTP verification.
func (api *TwoFAServiceAPI) Disable2FA(ctx context.Context, req *pb.Disable2FARequest) (*pb.Disable2FAResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if err := uuid.Validate(req.GetUserId()); err != nil {
		return nil, status.Error(codes.InvalidArgument, "user_id must be a valid UUID")
	}
	if req.GetOtpCode() == "" {
		return nil, status.Error(codes.InvalidArgument, "otp_code is required")
	}

	err := api.service.Disable(ctx, req.GetUserId(), req.GetOtpCode())
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotSetUp):
			return nil, status.Error(codes.FailedPrecondition, "2FA not set up")
		case errors.Is(err, domain.ErrNotEnabled):
			return nil, status.Error(codes.FailedPrecondition, "2FA not enabled")
		case errors.Is(err, domain.ErrRateLimitExceeded):
			return nil, status.Error(codes.ResourceExhausted, "too many verification attempts")
		case errors.Is(err, domain.ErrOTPReused):
			return nil, status.Error(codes.InvalidArgument, "OTP code already used")
		case errors.Is(err, domain.ErrInvalidOTP):
			return nil, status.Error(codes.InvalidArgument, "invalid OTP code")
		default:
			slog.Error("disable 2fa failed", "user_id", req.GetUserId(), "error", err)
			return nil, status.Error(codes.Internal, "disable failed")
		}
	}

	return &pb.Disable2FAResponse{}, nil
}
