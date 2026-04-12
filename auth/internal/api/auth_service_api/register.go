package auth_service_api

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vbncursed/vkr/auth/internal/domain"

	pb "github.com/vbncursed/vkr/auth/internal/pb/auth_api"
	pbmodels "github.com/vbncursed/vkr/auth/internal/pb/models"
)

// Register handles user registration.
func (api *AuthServiceAPI) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	user, accessToken, refreshToken, err := api.service.Register(ctx, req.Email, req.Password)
	if err != nil {
		var validErr *domain.PasswordValidationError
		if errors.As(err, &validErr) {
			return nil, status.Error(codes.InvalidArgument, validErr.Error())
		}
		if errors.Is(err, domain.ErrInvalidEmail) {
			return nil, status.Error(codes.InvalidArgument, "invalid email format")
		}
		if errors.Is(err, domain.ErrDuplicateEmail) {
			return nil, status.Error(codes.AlreadyExists, "user with this email already exists")
		}
		// Internal error -- do not leak details
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.RegisterResponse{
		Tokens: &pbmodels.TokenPair{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
		User: &pbmodels.User{
			Id:        user.ID,
			Email:     user.Email,
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
			UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
		},
	}, nil
}
