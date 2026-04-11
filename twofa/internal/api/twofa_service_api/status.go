package twofa_service_api

import (
	"context"

	pb "github.com/vbncursed/vkr/twofa/internal/pb/twofa_api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Get2FAStatus returns the current 2FA enrollment status for a user.
func (api *TwoFAServiceAPI) Get2FAStatus(ctx context.Context, req *pb.Get2FAStatusRequest) (*pb.Get2FAStatusResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Get2FAStatus not implemented")
}
