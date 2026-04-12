package auth_service_api

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/vbncursed/vkr/auth/internal/pb/auth_api"
)

// LogoutAll revokes all sessions for a given user.
// NOTE: This is an internal operation — the caller (Gateway) must authenticate
// the user via access token and extract user_id before forwarding the request.
// Direct calls to this RPC without prior authentication bypass authorization.
// SECURITY(WR-03): Caller authentication is deferred to Phase 9 (Gateway interceptors).
// Phase 9 will add a gRPC interceptor that validates a service-to-service token
// or mTLS certificate, ensuring only the Gateway can invoke internal RPCs.
func (api *AuthServiceAPI) LogoutAll(ctx context.Context, req *pb.LogoutAllRequest) (*pb.LogoutAllResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	if err := api.service.LogoutAll(ctx, req.UserId); err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.LogoutAllResponse{}, nil
}
