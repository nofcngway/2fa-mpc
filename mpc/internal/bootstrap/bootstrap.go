// Package bootstrap provides dependency injection factories for the MPC service.
package bootstrap

import (
	"context"
	"log/slog"

	"github.com/vbncursed/vkr/mpc/config"
	"github.com/vbncursed/vkr/mpc/internal/api/mpc_service_api"
	"github.com/vbncursed/vkr/mpc/internal/middleware"
	pb "github.com/vbncursed/vkr/mpc/internal/pb/mpc_api"
	"github.com/vbncursed/vkr/mpc/internal/services/mpcService"
	"github.com/vbncursed/vkr/mpc/internal/storage/pgstorage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// NewPGStorage creates a new PostgreSQL storage instance.
func NewPGStorage(ctx context.Context, cfg *config.Config) (*pgstorage.PGStorage, error) {
	storage, err := pgstorage.New(ctx, cfg.Database.DSN)
	if err != nil {
		return nil, err
	}
	slog.Info("PostgreSQL connected")
	return storage, nil
}

// NewMPCService creates a new MPC business logic service.
func NewMPCService(storage *pgstorage.PGStorage, cfg *config.Config) *mpcService.MPCService {
	return mpcService.NewMPCService(
		storage,
		[]byte(cfg.Node.EncryptionKey),
		cfg.Node.ID,
	)
}

// NewMPCServiceAPI creates a new gRPC handler for the MPC service.
func NewMPCServiceAPI(service *mpcService.MPCService) *mpc_service_api.MPCServiceAPI {
	return mpc_service_api.NewMPCServiceAPI(service)
}

// NewGRPCServer creates and configures a new gRPC server with interceptors and health check.
func NewGRPCServer(api *mpc_service_api.MPCServiceAPI) *grpc.Server {
	server := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.LoggingInterceptor),
	)

	pb.RegisterMPCNodeServiceServer(server, api)

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(server, healthServer)
	healthServer.SetServingStatus("mpc", healthpb.HealthCheckResponse_SERVING)

	slog.Info("gRPC server created with health check")
	return server
}
