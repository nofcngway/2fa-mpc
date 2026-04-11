package twofa_service_api

import (
	"context"

	pb "github.com/vbncursed/vkr/twofa/internal/pb/twofa_api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Verify2FA validates a TOTP code for a user.
func (api *TwoFAServiceAPI) Verify2FA(ctx context.Context, req *pb.Verify2FARequest) (*pb.Verify2FAResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Verify2FA not implemented")
}
