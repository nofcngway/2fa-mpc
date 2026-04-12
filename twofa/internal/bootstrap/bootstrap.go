package bootstrap

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/vbncursed/vkr/twofa/internal/pb/mpc_api"
	"github.com/vbncursed/vkr/twofa/config"
	"github.com/vbncursed/vkr/twofa/internal/api/twofa_service_api"
	"github.com/vbncursed/vkr/twofa/internal/middleware"
	pb "github.com/vbncursed/vkr/twofa/internal/pb/twofa_api"
	"github.com/vbncursed/vkr/twofa/internal/services/twofaService"
	"github.com/vbncursed/vkr/twofa/internal/storage/pgstorage"
	"github.com/vbncursed/vkr/twofa/internal/storage/redisstorage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
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

// NewMPCClients creates gRPC connections to all MPC nodes from config.
// Returns MPCClient slice (satisfying twofaService.MPCClient interface) and
// a slice of io.Closer for graceful shutdown of connections.
func NewMPCClients(cfg *config.Config) ([]twofaService.MPCClient, []io.Closer, error) {
	clients := make([]twofaService.MPCClient, len(cfg.MPCNodes))
	conns := make([]io.Closer, len(cfg.MPCNodes))

	for i, node := range cfg.MPCNodes {
		conn, err := grpc.NewClient(node.Addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithUnaryInterceptor(authMetadataInterceptor(cfg.SharedSecret)),
		)
		if err != nil {
			for j := 0; j < i; j++ {
				conns[j].Close()
			}
			return nil, nil, fmt.Errorf("connect to MPC node %d at %s: %w", i, node.Addr, err)
		}
		clients[i] = mpc_api.NewMPCNodeServiceClient(conn)
		conns[i] = conn
	}
	return clients, conns, nil
}

// authMetadataInterceptor returns a gRPC unary client interceptor that
// attaches the shared secret in "authorization" metadata on every outgoing call.
func authMetadataInterceptor(secret string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", secret)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// NewTwoFAService creates a new TwoFA business logic service.
func NewTwoFAService(
	storage *pgstorage.PGStorage,
	sessionStorage *redisstorage.RedisStorage,
	mpcClients []twofaService.MPCClient,
	cfg *config.Config,
) *twofaService.TwoFAService {
	return twofaService.NewTwoFAService(
		storage,
		sessionStorage,
		mpcClients,
		cfg.SharedSecret,
		cfg.GetMPCTimeout(),
	)
}

// NewTwoFAServiceAPI creates a new gRPC handler for TwoFA operations.
func NewTwoFAServiceAPI(service twofa_service_api.Service) *twofa_service_api.TwoFAServiceAPI {
	return twofa_service_api.NewTwoFAServiceAPI(service)
}

// NewGRPCServer creates and configures a new gRPC server with registered services.
func NewGRPCServer(api *twofa_service_api.TwoFAServiceAPI) *grpc.Server {
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
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
