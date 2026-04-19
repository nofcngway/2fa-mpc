// Package bootstrap provides dependency injection factories for the API gateway.
package bootstrap

import (
	"log/slog"
	"os"

	"github.com/vbncursed/vkr/gateway/config"
)

// SetupLogger creates and sets the default structured logger based on config.
func SetupLogger(cfg *config.Config) *slog.Logger {
	var level slog.Level
	switch cfg.Server.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}
