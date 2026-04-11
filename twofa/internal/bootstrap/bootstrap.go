package bootstrap

import (
	"context"
	"log/slog"

	"github.com/vbncursed/vkr/twofa/config"
	"github.com/vbncursed/vkr/twofa/internal/api/twofa_service_api"
	"github.com/vbncursed/vkr/twofa/internal/middleware"
	pb "github.com/vbncursed/vkr/twofa/internal/pb/twofa_api"
	"github.com/vbncursed/vkr/twofa/internal/services/twofaService"
	"github.com/vbncursed/vkr/twofa/internal/storage/pgstorage"
	"github.com/vbncursed/vkr/twofa/internal/storage/redisstorage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// NewPGStorage creates a new PostgreSQL storage instance.
func NewPGStorage(ctx context.Context, cfg *config.Config) (*pgstorage.PGStorage, error) {
	return pgstorage.NewPGStorage(ctx, cfg.Database.DSN)
}

// NewRedisStorage creates a new Redis storage instance.
func NewRedisStorage(ctx context.Context, cfg *config.Config) *redisstorage.RedisStorage {
	rs, err := redisstorage.NewRedisStorage(ctx, cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		slog.Warn("Redis unavailable, rate limiting disabled", "error", err)
		return nil
	}
	return rs
}

// NewTwoFAService creates a new TwoFA business logic service.
func NewTwoFAService(storage *pgstorage.PGStorage, sessionStorage *redisstorage.RedisStorage) *twofaService.TwoFAService {
	return twofaService.NewTwoFAService(storage, sessionStorage)
}

// NewTwoFAServiceAPI creates a new gRPC handler for TwoFA operations.
func NewTwoFAServiceAPI(service *twofaService.TwoFAService) *twofa_service_api.TwoFAServiceAPI {
	return twofa_service_api.NewTwoFAServiceAPI(service)
}

// NewGRPCServer creates and configures a new gRPC server with registered services.
func NewGRPCServer(api *twofa_service_api.TwoFAServiceAPI) *grpc.Server {
	server := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.LoggingInterceptor),
	)

	pb.RegisterTwoFAServiceServer(server, api)

	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(server, healthSrv)
	healthSrv.SetServingStatus("twofa.TwoFAService", healthpb.HealthCheckResponse_SERVING)

	return server
}
