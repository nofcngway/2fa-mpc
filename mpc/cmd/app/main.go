// Package main is the entry point for the MPC Node service.
package main

import (
	"log/slog"

	"github.com/vbncursed/vkr/mpc/config"
	"github.com/vbncursed/vkr/mpc/internal/bootstrap"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		slog.Error("failed to load config", "error", err)
		panic(err)
	}

	logger := bootstrap.SetupLogger(cfg)
	logger.Info("MPC Node starting", "node_id", cfg.Node.ID, "port", cfg.Server.Port)

	api, cleanup := bootstrap.InitServices(cfg, logger)
	defer cleanup()

	bootstrap.AppRun(api, cfg)
}
