package auth_service_api

import (
	"context"

	pb "github.com/vbncursed/vkr/auth/internal/pb/auth_api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Logout handles user session termination.
func (api *AuthServiceAPI) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}
