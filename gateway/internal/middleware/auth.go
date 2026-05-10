package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"google.golang.org/grpc/metadata"
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

// Auth wraps protected endpoints with JWT validation. The supplied
// IdentityResolver decides whether to consult a cache, hit Auth directly, or
// use a test stub — middleware itself is unaware of those concerns.
func Auth(resolver IdentityResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if publicPaths[r.URL.Path] || strings.HasPrefix(r.URL.Path, "/openapi/") {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			token, found := strings.CutPrefix(authHeader, "Bearer ")
			if !found || token == "" {
				writeUnauthorized(w, "missing or invalid authorization header")
				return
			}

			identity, err := resolver.Resolve(r.Context(), token)
			if err != nil {
				slog.Debug("token validation failed", "error", err)
				writeUnauthorized(w, "invalid or expired token")
				return
			}

			md := metadata.Pairs("x-user-id", identity.UserID, "x-user-email", identity.Email)
			ctx := metadata.NewOutgoingContext(r.Context(), md)
			ctx = context.WithValue(ctx, UserIDKey, identity.UserID)
			ctx = context.WithValue(ctx, EmailKey, identity.Email)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"code":16,"message":"` + msg + `"}`))
}
