package auth_service_api

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/vbncursed/vkr/auth/internal/pb/auth_api"
)

// ValidateToken verifies an access token and returns user claims.
func (api *AuthServiceAPI) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	if req.AccessToken == "" {
		return nil, status.Error(codes.InvalidArgument, "access token is required")
	}

	userID, email, err := api.service.ValidateToken(ctx, req.AccessToken)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	return &pb.ValidateTokenResponse{
		UserId: userID,
		Email:  email,
	}, nil
}
