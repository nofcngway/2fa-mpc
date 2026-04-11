package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/vbncursed/vkr/twofa/config"
	"github.com/vbncursed/vkr/twofa/internal/bootstrap"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load("config.yaml")
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	pgStorage, err := bootstrap.NewPGStorage(ctx, cfg)
	if err != nil {
		slog.Error("failed to create PostgreSQL storage", "error", err)
		os.Exit(1)
	}
	defer pgStorage.Close()

	redisStorage := bootstrap.NewRedisStorage(ctx, cfg)
	if redisStorage != nil {
		defer redisStorage.Close()
	}

	service := bootstrap.NewTwoFAService(pgStorage, redisStorage)
	api := bootstrap.NewTwoFAServiceAPI(service)
	grpcServer := bootstrap.NewGRPCServer(api)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		slog.Error("failed to listen", "port", cfg.Server.Port, "error", err)
		os.Exit(1)
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("TwoFA service started", "port", cfg.Server.Port)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("shutting down TwoFA service")
	grpcServer.GracefulStop()
	slog.Info("TwoFA service stopped")
}
