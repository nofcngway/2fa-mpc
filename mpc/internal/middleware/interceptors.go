// Package middleware provides gRPC interceptors for the MPC service.
package middleware

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthInterceptor creates a gRPC unary interceptor that validates a shared secret
// from the "authorization" metadata header using constant-time comparison.
// Health check requests (/grpc.health.v1.Health/Check) are excluded from authentication.
func AuthInterceptor(expectedSecret string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Exclude health check from authentication
		if info.FullMethod == "/grpc.health.v1.Health/Check" {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		values := md.Get("authorization")
		if len(values) == 0 || values[0] == "" {
			return nil, status.Error(codes.Unauthenticated, "missing authorization")
		}

		if subtle.ConstantTimeCompare([]byte(values[0]), []byte(expectedSecret)) != 1 {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization")
		}

		return handler(ctx, req)
	}
}

// LoggingInterceptor logs gRPC method calls with duration.
func LoggingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()

	resp, err := handler(ctx, req)

	slog.Info("gRPC call",
		"method", info.FullMethod,
		"duration", time.Since(start).String(),
		"error", err,
	)

	return resp, err
}
