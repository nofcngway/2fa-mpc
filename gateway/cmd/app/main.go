// Package main is the entry point for the API Gateway service.
package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/vbncursed/vkr/gateway/config"
	"github.com/vbncursed/vkr/gateway/internal/bootstrap"
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

	clients, err := bootstrap.NewGRPCClients(cfg)
	if err != nil {
		slog.Error("failed to create gRPC clients", "error", err)
		os.Exit(1)
	}

	rdb, err := bootstrap.NewRedisClient(ctx, cfg)
	if err != nil {
		slog.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}

	httpServer, err := bootstrap.NewHTTPServer(ctx, cfg, clients, rdb)
	if err != nil {
		slog.Error("failed to create HTTP server", "error", err)
		os.Exit(1)
	}

	metricsPort := cmp.Or(cfg.Server.MetricsPort, 9103)
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

	go func() {
		slog.Info("gateway started", "port", cfg.Server.Port)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server error", "error", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	slog.Info("shutting down gateway")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("failed to shutdown HTTP server", "error", err)
	}
	slog.Info("HTTP server stopped")

	if err := rdb.Close(); err != nil {
		slog.Error("failed to close Redis", "error", err)
	}
	slog.Info("Redis connection closed")

	clients.Close()
	slog.Info("gRPC connections closed")

	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("failed to shutdown metrics server", "error", err)
	}
	slog.Info("metrics server stopped")

	slog.Info("gateway shutdown complete")
}
