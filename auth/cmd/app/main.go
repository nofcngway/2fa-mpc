package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/vbncursed/vkr/auth/config"
	"github.com/vbncursed/vkr/auth/internal/bootstrap"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// slog JSON handler with configurable log level
	logLevel := slog.LevelInfo
	switch cfg.Server.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pgStorage, err := bootstrap.NewPGStorage(ctx, cfg)
	if err != nil {
		slog.Error("failed to connect to PostgreSQL", "error", err)
		os.Exit(1)
	}

	redisStorage, err := bootstrap.NewRedisStorage(ctx, cfg)
	if err != nil {
		slog.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}

	authSvc, err := bootstrap.NewAuthService(cfg, pgStorage, redisStorage)
	if err != nil {
		slog.Error("failed to create auth service", "error", err)
		os.Exit(1)
	}
	authAPI := bootstrap.NewAuthServiceAPI(authSvc)
	grpcServer := bootstrap.NewGRPCServer(authAPI)

	// Metrics HTTP server on separate port
	metricsPort := cfg.Server.MetricsPort
	if metricsPort == 0 {
		metricsPort = 9100
	}
	metricsServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", metricsPort),
		Handler: promhttp.Handler(),
	}
	go func() {
		slog.Info("metrics server started", "port", metricsPort)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("metrics server error", "error", err)
		}
	}()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		slog.Error("failed to listen", "port", cfg.Server.Port, "error", err)
		os.Exit(1)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("auth service started", "port", cfg.Server.Port)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC server error", "error", err)
		}
	}()

	<-sigCh
	slog.Info("shutting down auth service")

	// Ordered shutdown with 30s timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// 1. Stop accepting new gRPC requests
	grpcServer.GracefulStop()
	slog.Info("gRPC server stopped")

	// 2. Close Redis
	if redisStorage != nil {
		redisStorage.Close()
		slog.Info("Redis connection closed")
	}

	// 3. Close PostgreSQL
	pgStorage.Close()
	slog.Info("PostgreSQL connection closed")

	// 4. Shutdown metrics HTTP server
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("failed to shutdown metrics server", "error", err)
	}
	slog.Info("metrics server stopped")

	cancel()
	slog.Info("auth service shutdown complete")
}
