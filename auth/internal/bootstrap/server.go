package bootstrap

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/vbncursed/vkr/auth/internal/api/auth_service_api"
	"github.com/vbncursed/vkr/auth/internal/middleware"
	pb "github.com/vbncursed/vkr/auth/internal/pb/auth_api"
)

// NewGRPCServer creates and configures a gRPC server with interceptors and health check.
func NewGRPCServer(api *auth_service_api.AuthServiceAPI) *grpc.Server {
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.RecoveryInterceptor,
			middleware.MetricsInterceptor,
			middleware.LoggingInterceptor,
		),
	)

	pb.RegisterAuthServiceServer(server, api)

	healthServer := health.NewServer()
	healthServer.SetServingStatus("auth", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, healthServer)

	return server
}
