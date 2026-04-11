package mpc_service_api

import (
	"context"

	pb "github.com/vbncursed/vkr/mpc/internal/pb/mpc_api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RetrieveShare handles the RetrieveShare RPC call.
func (api *MPCServiceAPI) RetrieveShare(ctx context.Context, req *pb.RetrieveShareRequest) (*pb.RetrieveShareResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}
