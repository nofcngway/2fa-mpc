// Package main is the entry point for the MPC Node service.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/vbncursed/vkr/mpc/config"
	"github.com/vbncursed/vkr/mpc/internal/bootstrap"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.Load("config.yaml")
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	slog.Info("MPC Node starting", "node_id", cfg.Node.ID, "port", cfg.Server.Port)

	storage, err := bootstrap.NewPGStorage(ctx, cfg)
	if err != nil {
		slog.Error("failed to connect to PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer storage.Close()

	service, err := bootstrap.NewMPCService(storage, cfg)
	if err != nil {
		slog.Error("failed to create MPC service", "error", err)
		os.Exit(1)
	}
	api := bootstrap.NewMPCServiceAPI(service)
	grpcServer := bootstrap.NewGRPCServer(api, cfg)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		slog.Info("shutting down MPC Node...")
		grpcServer.GracefulStop()
		cancel()
	}()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		slog.Error("failed to listen", "port", cfg.Server.Port, "error", err)
		os.Exit(1)
	}

	slog.Info("MPC Node listening", "port", cfg.Server.Port)
	if err := grpcServer.Serve(lis); err != nil {
		slog.Error("gRPC server error", "error", err)
		os.Exit(1)
	}
}
