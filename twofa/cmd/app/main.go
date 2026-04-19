// Package main is the entry point for the TwoFA service.
package main

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

	"github.com/vbncursed/vkr/twofa/config"
	"github.com/vbncursed/vkr/twofa/internal/bootstrap"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	slog.SetDefault(bootstrap.NewLogger(cfg))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pgStorage, err := bootstrap.NewPGStorage(ctx, cfg)
	if err != nil {
		slog.Error("failed to create PostgreSQL storage", "error", err)
		os.Exit(1)
	}

	sessionStorage := bootstrap.NewSessionStorage(ctx, cfg)

	mpcClients, mpcConns, err := bootstrap.NewMPCClients(cfg)
	if err != nil {
		slog.Error("failed to create MPC clients", "error", err)
		os.Exit(1)
	}

	kafkaProducer := bootstrap.NewKafkaProducer(cfg.Kafka.Brokers, cfg.Kafka.Topic)

	service, err := bootstrap.NewTwoFAService(pgStorage, sessionStorage, mpcClients, kafkaProducer, cfg)
	if err != nil {
		slog.Error("failed to create TwoFA service", "error", err)
		os.Exit(1)
	}
	api := bootstrap.NewTwoFAServiceAPI(service)
	grpcServer := bootstrap.NewGRPCServer(api)

	// Metrics HTTP server on separate port
	metricsPort := cmp.Or(cfg.Server.MetricsPort, 9101)
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

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		slog.Error("failed to listen", "port", cfg.Server.Port, "error", err)
		os.Exit(1)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("TwoFA service started", "port", cfg.Server.Port)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC server failed", "error", err)
		}
	}()

	<-quit
	slog.Info("shutting down TwoFA service")

	// Ordered shutdown with 30s timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// 1. Stop gRPC with timeout fallback
	grpcDone := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(grpcDone)
	}()
	select {
	case <-grpcDone:
		slog.Info("gRPC server stopped gracefully")
	case <-shutdownCtx.Done():
		grpcServer.Stop()
		slog.Warn("gRPC server force-stopped after timeout")
	}

	// 2. Flush Kafka (pending audit events)
	if err := kafkaProducer.Close(); err != nil {
		slog.Error("failed to close Kafka producer", "error", err)
	}
	slog.Info("Kafka producer closed")

	// 3. Close Redis (if backed by real Redis, not NoOp)
	if closer, ok := sessionStorage.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			slog.Error("failed to close Redis", "error", err)
		}
		slog.Info("Redis connection closed")
	}

	// 4. Close MPC connections
	for i, conn := range mpcConns {
		if err := conn.Close(); err != nil {
			slog.Error("failed to close MPC connection", "index", i, "error", err)
		}
	}
	slog.Info("MPC connections closed")

	// 5. Close PostgreSQL
	pgStorage.Close()
	slog.Info("PostgreSQL connection closed")

	// 6. Shutdown metrics HTTP server
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("failed to shutdown metrics server", "error", err)
	}
	slog.Info("metrics server stopped")

	slog.Info("TwoFA service shutdown complete")
}
