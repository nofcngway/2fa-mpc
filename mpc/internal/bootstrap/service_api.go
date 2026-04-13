package bootstrap

import (
	"github.com/vbncursed/vkr/mpc/internal/api/mpc_service_api"
)

// NewMPCServiceAPI creates a new gRPC handler for the MPC service.
func NewMPCServiceAPI(service mpc_service_api.Service) *mpc_service_api.MPCServiceAPI {
	return mpc_service_api.NewMPCServiceAPI(service)
}
