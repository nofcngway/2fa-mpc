package bootstrap

import (
	"github.com/vbncursed/vkr/twofa/internal/api/twofa_service_api"
)

// NewTwoFAServiceAPI creates a new gRPC handler for TwoFA operations.
func NewTwoFAServiceAPI(service twofa_service_api.Service) *twofa_service_api.TwoFAServiceAPI {
	return twofa_service_api.NewTwoFAServiceAPI(service)
}
