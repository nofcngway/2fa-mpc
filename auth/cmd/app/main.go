package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/vbncursed/vkr/auth/config"
	"github.com/vbncursed/vkr/auth/internal/bootstrap"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.Load("config.yaml")
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	pgStorage, err := bootstrap.NewPGStorage(ctx, cfg)
	if err != nil {
		slog.Error("failed to connect to PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer pgStorage.Close()

	redisStorage, err := bootstrap.NewRedisStorage(ctx, cfg)
	if err != nil {
		slog.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer redisStorage.Close()

	authSvc, err := bootstrap.NewAuthService(cfg, pgStorage, redisStorage)
	if err != nil {
		slog.Error("failed to create auth service", "error", err)
		os.Exit(1)
	}
	authAPI := bootstrap.NewAuthServiceAPI(authSvc)
	grpcServer := bootstrap.NewGRPCServer(authAPI)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		slog.Info("shutting down auth service")
		grpcServer.GracefulStop()
		cancel()
	}()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		slog.Error("failed to listen", "port", cfg.Server.Port, "error", err)
		os.Exit(1)
	}

	slog.Info("auth service started", "port", cfg.Server.Port)

	if err := grpcServer.Serve(lis); err != nil {
		slog.Error("gRPC server error", "error", err)
		os.Exit(1)
	}
}
