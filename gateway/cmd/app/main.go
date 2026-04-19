// Package main is the entry point for the API Gateway service.
package main

import (
	"log/slog"

	"github.com/vbncursed/vkr/gateway/config"
	"github.com/vbncursed/vkr/gateway/internal/bootstrap"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		slog.Error("failed to load config", "error", err)
		panic(err)
	}

	logger := bootstrap.SetupLogger(cfg)
	slog.SetDefault(logger)
	logger.Info("starting gateway", "port", cfg.Server.Port)

	httpServer, cleanup := bootstrap.InitServices(cfg, logger)
	defer cleanup()

	bootstrap.AppRun(httpServer, cfg)
}
