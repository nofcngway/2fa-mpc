package auth_service_api

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vbncursed/vkr/auth/internal/domain"

	pb "github.com/vbncursed/vkr/auth/internal/pb/auth_api"
	pbmodels "github.com/vbncursed/vkr/auth/internal/pb/models"
)

// RefreshToken handles JWT token refresh using a valid refresh token.
func (api *AuthServiceAPI) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	if req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token is required")
	}

	accessToken, refreshToken, err := api.service.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidToken) {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}
		if errors.Is(err, domain.ErrTokenExpired) {
			return nil, status.Error(codes.Unauthenticated, "token expired")
		}
		if errors.Is(err, domain.ErrTokenRevoked) {
			return nil, status.Error(codes.Unauthenticated, "token revoked")
		}
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.RefreshTokenResponse{
		Tokens: &pbmodels.TokenPair{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
	}, nil
}
