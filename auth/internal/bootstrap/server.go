// Package bootstrap wires application dependencies and constructs the gRPC server.
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

	"github.com/vbncursed/vkr/auth/config"
	"github.com/vbncursed/vkr/auth/internal/api/auth_service_api"
	"github.com/vbncursed/vkr/auth/internal/middleware"
	pb "github.com/vbncursed/vkr/auth/internal/pb/auth_api"
)

// NewGRPCServer creates and configures a gRPC server with interceptors and health check.
func NewGRPCServer(api *auth_service_api.AuthServiceAPI, cfg *config.Config) (*grpc.Server, error) {
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(4 * 1024 * 1024), // 4 MB — auth payloads are small
		grpc.MaxSendMsgSize(4 * 1024 * 1024),
		grpc.ChainUnaryInterceptor(
			middleware.RecoveryInterceptor,
			middleware.MetricsInterceptor,
			middleware.LoggingInterceptor,
		),
	}

	if cfg.TLS.Enabled {
		creds, err := loadServerTLSCredentials(cfg.TLS.CertFile, cfg.TLS.KeyFile, cfg.TLS.CAFile)
		if err != nil {
			return nil, fmt.Errorf("load tls credentials: %w", err)
		}
		opts = append(opts, grpc.Creds(creds))
		slog.Info("mTLS enabled", "service", "auth", "cert", cfg.TLS.CertFile)
	} else {
		slog.Warn("mTLS DISABLED — running in insecure mode", "service", "auth")
	}

	server := grpc.NewServer(opts...)

	pb.RegisterAuthServiceServer(server, api)

	healthServer := health.NewServer()
	healthServer.SetServingStatus("auth", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, healthServer)

	return server, nil
}

// AppRun starts the gRPC and metrics servers, then blocks until a shutdown signal
// is received. It performs graceful shutdown with a 30-second timeout.
func AppRun(api *auth_service_api.AuthServiceAPI, cfg *config.Config) {
	// 1. Create gRPC server
	grpcServer, err := NewGRPCServer(api, cfg)
	if err != nil {
		slog.Error("failed to create gRPC server", "error", err)
		os.Exit(1)
	}

	// 2. Create listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		slog.Error("failed to listen", "port", cfg.Server.Port, "error", err)
		os.Exit(1)
	}

	// 3. Metrics HTTP server on separate port
	metricsPort := cmp.Or(cfg.Server.MetricsPort, 9100)
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
		slog.Info("auth service started", "port", cfg.Server.Port)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC server error", "error", err)
		}
	}()

	// 5. Block on signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	slog.Info("shutting down auth service")

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

	slog.Info("auth service shutdown complete")
}
