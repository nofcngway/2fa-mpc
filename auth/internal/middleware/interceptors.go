package middleware

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RecoveryInterceptor catches panics in handlers and returns codes.Internal
// instead of crashing the process.
func RecoveryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic recovered in gRPC handler",
				"method", info.FullMethod,
				"panic", r,
				"stack", string(debug.Stack()),
			)
			err = status.Errorf(codes.Internal, "internal error")
		}
	}()
	return handler(ctx, req)
}

// MetricsInterceptor records gRPC request count and duration.
func MetricsInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	duration := time.Since(start).Seconds()

	st, _ := status.FromError(err)
	grpcRequestsTotal.WithLabelValues(info.FullMethod, st.Code().String()).Inc()
	grpcRequestDuration.WithLabelValues(info.FullMethod).Observe(duration)

	return resp, err
}

// LoggingInterceptor logs gRPC calls with method, duration, and error (if any).
func LoggingInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	slog.Info("gRPC call",
		"method", info.FullMethod,
		"duration", time.Since(start).String(),
		"error", err,
	)
	return resp, err
}
