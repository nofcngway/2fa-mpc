// Package main is the entry point for the MPC Node service.
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

	"github.com/vbncursed/vkr/mpc/config"
	"github.com/vbncursed/vkr/mpc/internal/bootstrap"
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

	slog.Info("MPC Node starting", "node_id", cfg.Node.ID, "port", cfg.Server.Port)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	storage, err := bootstrap.NewPGStorage(ctx, cfg)
	if err != nil {
		slog.Error("failed to connect to PostgreSQL", "error", err)
		os.Exit(1)
	}

	kafkaProducer := bootstrap.NewKafkaProducer(cfg.Kafka.Brokers, cfg.Kafka.Topic)

	service, err := bootstrap.NewMPCService(storage, cfg, kafkaProducer)
	if err != nil {
		slog.Error("failed to create MPC service", "error", err)
		os.Exit(1)
	}
	api := bootstrap.NewMPCServiceAPI(service)
	grpcServer := bootstrap.NewGRPCServer(api, cfg)

	// Metrics HTTP server on separate port
	metricsPort := cfg.Server.MetricsPort
	if metricsPort == 0 {
		metricsPort = 9102
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
		slog.Info("MPC Node listening", "port", cfg.Server.Port)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC server error", "error", err)
		}
	}()

	<-sigCh
	slog.Info("shutting down MPC Node")

	// Ordered shutdown with 30s timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// 1. Stop gRPC
	grpcServer.GracefulStop()
	slog.Info("gRPC server stopped")

	// 2. Flush Kafka (pending audit events)
	if err := kafkaProducer.Close(); err != nil {
		slog.Error("failed to close Kafka producer", "error", err)
	}
	slog.Info("Kafka producer closed")

	// 3. Close PostgreSQL
	storage.Close()
	slog.Info("PostgreSQL connection closed")

	// 4. Shutdown metrics HTTP server
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("failed to shutdown metrics server", "error", err)
	}
	slog.Info("metrics server stopped")

	cancel()
	slog.Info("MPC Node shutdown complete")
}
