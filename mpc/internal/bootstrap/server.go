package bootstrap

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/vbncursed/vkr/mpc/config"
	"github.com/vbncursed/vkr/mpc/internal/api/mpc_service_api"
	"github.com/vbncursed/vkr/mpc/internal/middleware"
	pb "github.com/vbncursed/vkr/mpc/internal/pb/mpc_api"
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

	return server
}

// AppRun starts the gRPC and metrics servers, then blocks until a shutdown signal
// is received. It performs graceful shutdown with a 30-second timeout.
func AppRun(api *mpc_service_api.MPCServiceAPI, cfg *config.Config) {
	// 1. Create gRPC server
	grpcServer := NewGRPCServer(api, cfg)

	// 2. Create listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		slog.Error("failed to listen", "port", cfg.Server.Port, "error", err)
		os.Exit(1)
	}

	// 3. Metrics HTTP server on separate port
	metricsPort := cmp.Or(cfg.Server.MetricsPort, 9102)
	metricsServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", metricsPort),
		Handler:      promhttp.Handler(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	go func() {
		slog.Info("metrics server started", "port", metricsPort)
		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("metrics server error", "error", err)
		}
	}()

	// 4. Start gRPC
	go func() {
		slog.Info("MPC Node listening", "port", cfg.Server.Port)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC server error", "error", err)
		}
	}()

	// 5. Block on signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	slog.Info("shutting down MPC Node")

	// 6. Graceful stop with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	grpcDone := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(grpcDone)
	}()
	select {
	case <-grpcDone:
	case <-shutdownCtx.Done():
		grpcServer.Stop()
	}
	slog.Info("gRPC server stopped")

	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("failed to shutdown metrics server", "error", err)
	}
	slog.Info("metrics server stopped")

	slog.Info("MPC Node shutdown complete")
}
