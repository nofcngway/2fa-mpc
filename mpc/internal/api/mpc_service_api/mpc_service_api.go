// Package mpc_service_api provides gRPC handlers for the MPC Node service.
package mpc_service_api

import (
	pb "github.com/vbncursed/vkr/mpc/internal/pb/mpc_api"
	"github.com/vbncursed/vkr/mpc/internal/services/mpcService"
)

// MPCServiceAPI implements the gRPC MPCNodeServiceServer interface.
type MPCServiceAPI struct {
	pb.UnimplementedMPCNodeServiceServer
	service *mpcService.MPCService
}

// NewMPCServiceAPI creates a new MPCServiceAPI instance.
func NewMPCServiceAPI(service *mpcService.MPCService) *MPCServiceAPI {
	return &MPCServiceAPI{
		service: service,
	}
}
