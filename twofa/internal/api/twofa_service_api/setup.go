package twofa_service_api

import (
	"context"

	pb "github.com/vbncursed/vkr/twofa/internal/pb/twofa_api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Setup2FA initiates 2FA enrollment for a user.
func (api *TwoFAServiceAPI) Setup2FA(ctx context.Context, req *pb.Setup2FARequest) (*pb.Setup2FAResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Setup2FA not implemented")
}
