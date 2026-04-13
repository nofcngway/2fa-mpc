package bootstrap

import (
	"github.com/vbncursed/vkr/twofa/internal/api/twofa_service_api"
	"github.com/vbncursed/vkr/twofa/internal/middleware"
	pb "github.com/vbncursed/vkr/twofa/internal/pb/twofa_api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// NewGRPCServer creates and configures a new gRPC server with registered services.
func NewGRPCServer(api *twofa_service_api.TwoFAServiceAPI) *grpc.Server {
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.RecoveryInterceptor,
			middleware.MetricsInterceptor,
			middleware.LoggingInterceptor,
		),
	)

	pb.RegisterTwoFAServiceServer(server, api)

	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(server, healthSrv)
	healthSrv.SetServingStatus("twofa.TwoFAService", healthpb.HealthCheckResponse_SERVING)

	return server
}
