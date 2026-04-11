package twofa_service_api

import (
	"context"

	pb "github.com/vbncursed/vkr/twofa/internal/pb/twofa_api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Disable2FA removes 2FA enrollment for a user.
func (api *TwoFAServiceAPI) Disable2FA(ctx context.Context, req *pb.Disable2FARequest) (*pb.Disable2FAResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Disable2FA not implemented")
}
