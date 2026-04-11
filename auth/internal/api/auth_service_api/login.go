package auth_service_api

import (
	"context"

	pb "github.com/vbncursed/vkr/auth/internal/pb/auth_api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Login handles user authentication.
func (api *AuthServiceAPI) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}
