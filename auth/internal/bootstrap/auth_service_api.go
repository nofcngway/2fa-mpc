package bootstrap

import (
	"github.com/vbncursed/vkr/auth/internal/api/auth_service_api"
)

// NewAuthServiceAPI creates a new gRPC AuthServiceAPI handler.
func NewAuthServiceAPI(service auth_service_api.Service) *auth_service_api.AuthServiceAPI {
	return auth_service_api.NewAuthServiceAPI(service)
}
