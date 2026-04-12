package twofa_service_api

import (
	"context"
	"errors"

	pb "github.com/vbncursed/vkr/twofa/internal/pb/twofa_api"
	"github.com/vbncursed/vkr/twofa/internal/services/twofaService"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Setup2FA initiates 2FA enrollment for a user.
// Validates input, delegates to service layer, maps domain errors to gRPC status codes.
func (api *TwoFAServiceAPI) Setup2FA(ctx context.Context, req *pb.Setup2FARequest) (*pb.Setup2FAResponse, error) {
	if req.UserId == "" || req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and email are required")
	}

	uri, backupCodes, err := api.service.Setup(ctx, req.UserId, req.Email)
	if err != nil {
		if errors.Is(err, twofaService.ErrAlreadyEnabled) {
			return nil, status.Error(codes.AlreadyExists, "2FA already enabled")
		}
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.Setup2FAResponse{
		ProvisioningUri: uri,
		BackupCodes:     backupCodes,
	}, nil
}
