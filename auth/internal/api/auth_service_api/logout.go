package auth_service_api

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vbncursed/vkr/auth/internal/domain"

	pb "github.com/vbncursed/vkr/auth/internal/pb/auth_api"
)

// Logout handles user session termination.
func (api *AuthServiceAPI) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	if req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token is required")
	}

	if err := api.service.Logout(ctx, req.RefreshToken); err != nil {
		if errors.Is(err, domain.ErrInvalidToken) || errors.Is(err, domain.ErrTokenExpired) {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.LogoutResponse{}, nil
}
