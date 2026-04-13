package bootstrap

import (
	"log/slog"
	"os"

	"github.com/vbncursed/vkr/auth/config"
)

// NewLogger creates a JSON slog.Logger with the log level from config.
func NewLogger(cfg *config.Config) *slog.Logger {
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
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
}
