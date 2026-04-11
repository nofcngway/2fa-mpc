package auth_service_api

import (
	"github.com/vbncursed/vkr/auth/internal/services/authService"

	pb "github.com/vbncursed/vkr/auth/internal/pb/auth_api"
)

// AuthServiceAPI implements the gRPC AuthService interface.
type AuthServiceAPI struct {
	pb.UnimplementedAuthServiceServer
	service *authService.AuthService
}

// NewAuthServiceAPI creates a new AuthServiceAPI.
func NewAuthServiceAPI(service *authService.AuthService) *AuthServiceAPI {
	return &AuthServiceAPI{service: service}
}
