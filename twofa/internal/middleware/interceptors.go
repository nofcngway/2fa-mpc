package middleware

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
)

// LoggingInterceptor logs gRPC method calls with duration.
func LoggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()

	resp, err := handler(ctx, req)

	duration := time.Since(start)
	slog.Info("gRPC call",
		"method", info.FullMethod,
		"duration", duration.String(),
		"error", err,
	)

	return resp, err
}
