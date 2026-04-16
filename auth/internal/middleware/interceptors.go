// Package middleware provides gRPC unary interceptors for logging, metrics, and panic recovery.
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

// LoggingInterceptor logs gRPC calls with method, duration, and status code.
// Raw errors are not logged to avoid leaking sensitive data (tokens, passwords).
func LoggingInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)

	st, _ := status.FromError(err)
	attrs := []any{
		"method", info.FullMethod,
		"duration", time.Since(start).String(),
		"code", st.Code().String(),
	}
	if err != nil {
		// Log only the gRPC status message, which is controlled by our handlers
		// and never contains raw internal errors or sensitive data.
		attrs = append(attrs, "status_msg", st.Message())
		slog.Warn("gRPC call failed", attrs...)
	} else {
		slog.Info("gRPC call", attrs...)
	}

	return resp, err
}
