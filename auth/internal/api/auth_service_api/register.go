package auth_service_api

import (
	"context"

	pb "github.com/vbncursed/vkr/auth/internal/pb/auth_api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Register handles user registration.
func (api *AuthServiceAPI) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}
