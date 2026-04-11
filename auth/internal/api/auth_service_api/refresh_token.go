package auth_service_api

import (
	"context"

	pb "github.com/vbncursed/vkr/auth/internal/pb/auth_api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RefreshToken handles JWT token refresh using a valid refresh token.
func (api *AuthServiceAPI) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}
