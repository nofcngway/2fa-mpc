package bootstrap

import (
	"log/slog"
	"os"

	"github.com/vbncursed/vkr/twofa/config"
)

// NewLogger creates a JSON slog.Logger with the log level from config.
func NewLogger(cfg *config.Config) *slog.Logger {
	logLevel := slog.LevelInfo
	switch cfg.Server.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
}
