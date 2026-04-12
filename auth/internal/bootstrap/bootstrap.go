package bootstrap

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/vbncursed/vkr/auth/config"
	"github.com/vbncursed/vkr/auth/internal/api/auth_service_api"
	pb "github.com/vbncursed/vkr/auth/internal/pb/auth_api"
	"github.com/vbncursed/vkr/auth/internal/middleware"
	"github.com/vbncursed/vkr/auth/internal/services/authService"
	"github.com/vbncursed/vkr/auth/internal/storage/pgstorage"
	"github.com/vbncursed/vkr/auth/internal/storage/redisstorage"
)

// NewPGStorage creates a new PostgreSQL storage connection.
func NewPGStorage(ctx context.Context, cfg *config.Config) (*pgstorage.PGStorage, error) {
	storage, err := pgstorage.New(ctx, cfg.Database.DSN)
	if err != nil {
		return nil, err
	}
	slog.Info("PostgreSQL connected")
	return storage, nil
}

// NewRedisStorage creates a new Redis storage connection.
// Returns an error if Redis is unreachable — session operations require Redis.
func NewRedisStorage(ctx context.Context, cfg *config.Config) (*redisstorage.RedisStorage, error) {
	rs := redisstorage.New(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err := rs.Ping(ctx); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}
	slog.Info("Redis connected")
	return rs, nil
}

// NewAuthService creates a new AuthService with the provided storage dependencies and RSA keys.
func NewAuthService(cfg *config.Config, storage authService.Storage, sessionStorage authService.SessionStorage) (*authService.AuthService, error) {
	privateKey, publicKey, err := authService.LoadRSAKeys(cfg.JWT.PrivateKeyPath, cfg.JWT.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load RSA keys: %w", err)
	}

	return authService.NewAuthService(
		storage, sessionStorage,
		privateKey, publicKey,
		cfg.JWT.AccessTokenTTL, cfg.JWT.RefreshTokenTTL,
	), nil
}

// NewAuthServiceAPI creates a new gRPC AuthServiceAPI handler.
func NewAuthServiceAPI(service auth_service_api.Service) *auth_service_api.AuthServiceAPI {
	return auth_service_api.NewAuthServiceAPI(service)
}

// NewGRPCServer creates and configures a gRPC server with interceptors and health check.
func NewGRPCServer(api *auth_service_api.AuthServiceAPI) *grpc.Server {
	server := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.LoggingInterceptor),
	)

	pb.RegisterAuthServiceServer(server, api)

	healthServer := health.NewServer()
	healthServer.SetServingStatus("auth", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, healthServer)

	return server
}
