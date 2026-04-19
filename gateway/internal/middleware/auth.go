package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"google.golang.org/grpc/metadata"

	pb "github.com/vbncursed/vkr/gateway/internal/pb/auth_api"
)

type contextKey string

const (
	UserIDKey contextKey = "user_id"
	EmailKey  contextKey = "email"
)

var publicPaths = map[string]bool{
	"/api/v1/auth/register": true,
	"/api/v1/auth/login":    true,
	"/api/v1/auth/refresh":  true,
	"/api/v1/auth/validate": true,
	"/docs":                 true,
	"/healthz":              true,
}

func Auth(authClient pb.AuthServiceClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if publicPaths[r.URL.Path] || strings.HasPrefix(r.URL.Path, "/openapi/") {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			token, found := strings.CutPrefix(authHeader, "Bearer ")
			if !found || token == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"code":16,"message":"missing or invalid authorization header"}`))
				return
			}

			resp, err := authClient.ValidateToken(r.Context(), &pb.ValidateTokenRequest{AccessToken: token})
			if err != nil {
				slog.Debug("token validation failed", "error", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"code":16,"message":"invalid or expired token"}`))
				return
			}

			md := metadata.Pairs("x-user-id", resp.UserId, "x-user-email", resp.Email)
			ctx := metadata.NewOutgoingContext(r.Context(), md)
			ctx = context.WithValue(ctx, UserIDKey, resp.UserId)
			ctx = context.WithValue(ctx, EmailKey, resp.Email)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
