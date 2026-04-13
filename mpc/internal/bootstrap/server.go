package bootstrap

import (
	"log/slog"

	"github.com/vbncursed/vkr/mpc/config"
	"github.com/vbncursed/vkr/mpc/internal/api/mpc_service_api"
	"github.com/vbncursed/vkr/mpc/internal/middleware"
	pb "github.com/vbncursed/vkr/mpc/internal/pb/mpc_api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// NewGRPCServer creates and configures a new gRPC server with auth + logging interceptors and health check.
func NewGRPCServer(api *mpc_service_api.MPCServiceAPI, cfg *config.Config) *grpc.Server {
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.MetricsInterceptor,
			middleware.AuthInterceptor(cfg.SharedSecret),
			middleware.LoggingInterceptor,
		),
	)

	pb.RegisterMPCNodeServiceServer(server, api)

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(server, healthServer)
	healthServer.SetServingStatus("mpc", healthpb.HealthCheckResponse_SERVING)

	slog.Info("gRPC server created with health check")
	return server
}
