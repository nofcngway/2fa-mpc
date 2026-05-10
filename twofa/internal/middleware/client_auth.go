package middleware

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// ClientAuthInterceptor returns a gRPC unary client interceptor that attaches
// the shared secret in "authorization" metadata on every outgoing call. It is
// retained as defense-in-depth on top of mTLS — if mTLS is misconfigured, the
// downstream server still rejects unauthenticated requests via its
// AuthInterceptor.
func ClientAuthInterceptor(secret string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", secret)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
