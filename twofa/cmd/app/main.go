// Package main is the entry point for the TwoFA service.
package main

import (
	"log/slog"

	"github.com/vbncursed/vkr/twofa/config"
	"github.com/vbncursed/vkr/twofa/internal/bootstrap"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		slog.Error("failed to load config", "error", err)
		panic(err)
	}

	logger := bootstrap.SetupLogger(cfg)
	logger.Info("starting TwoFA service", "port", cfg.Server.Port)

	api, cleanup := bootstrap.InitServices(cfg, logger)
	defer cleanup()

	bootstrap.AppRun(api, cfg)
}
