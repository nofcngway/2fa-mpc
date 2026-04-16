// Package auth_service_api provides the gRPC transport layer for authentication operations.
package auth_service_api

import (
	"context"

	"github.com/vbncursed/vkr/auth/internal/domain"

	pb "github.com/vbncursed/vkr/auth/internal/pb/auth_api"
)

// Service defines the contract the API layer requires from the auth service.
type Service interface {
	Register(ctx context.Context, email, password string) (*domain.User, string, string, error)
	Login(ctx context.Context, email, password string) (*domain.User, string, string, error)
	Logout(ctx context.Context, refreshTokenStr string) error
	LogoutAll(ctx context.Context, userID string) error
	RefreshToken(ctx context.Context, refreshTokenStr string) (string, string, error)
	ValidateToken(ctx context.Context, accessTokenStr string) (string, string, error)
}

// Compile-time check: AuthServiceAPI must implement pb.AuthServiceServer.
var _ pb.AuthServiceServer = (*AuthServiceAPI)(nil)

// AuthServiceAPI implements the gRPC AuthService interface.
type AuthServiceAPI struct {
	pb.UnimplementedAuthServiceServer
	service Service
}

// NewAuthServiceAPI creates a new AuthServiceAPI.
func NewAuthServiceAPI(service Service) *AuthServiceAPI {
	return &AuthServiceAPI{service: service}
}
