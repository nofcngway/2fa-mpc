package auth_service_api

import (
	"context"

	pb "github.com/vbncursed/vkr/auth/internal/pb/auth_api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ValidateToken verifies an access token and returns user claims.
func (api *AuthServiceAPI) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}
