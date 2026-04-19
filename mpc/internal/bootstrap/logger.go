package bootstrap

import (
	"log/slog"
	"os"

	"github.com/vbncursed/vkr/mpc/config"
)

// SetupLogger creates a JSON slog.Logger with the log level from config,
// sets it as the default logger, and returns it.
func SetupLogger(cfg *config.Config) *slog.Logger {
	logLevel := slog.LevelInfo
	switch cfg.Server.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		slog.Warn("unknown log level, defaulting to info", "level", cfg.Server.LogLevel)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)
	return logger
}
