package middleware

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// MetricsInterceptor records gRPC request count and duration.
func MetricsInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	duration := time.Since(start).Seconds()

	st, _ := status.FromError(err)
	grpcRequestsTotal.WithLabelValues(info.FullMethod, st.Code().String()).Inc()
	grpcRequestDuration.WithLabelValues(info.FullMethod).Observe(duration)

	return resp, err
}

// LoggingInterceptor logs gRPC calls with method, duration, and error (if any).
func LoggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	slog.Info("gRPC call",
		"method", info.FullMethod,
		"duration", time.Since(start).String(),
		"error", err,
	)
	return resp, err
}
