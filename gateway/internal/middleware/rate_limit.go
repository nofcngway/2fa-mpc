package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

func RateLimit(rdb *redis.Client, requestsPerMinute, burst int) func(http.Handler) http.Handler {
	window := time.Minute
	limit := int64(requestsPerMinute + burst)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			key := fmt.Sprintf("gateway:rate_limit:%s", ip)

			count, err := rdb.Incr(r.Context(), key).Result()
			if err != nil {
				slog.Warn("rate limit check failed, allowing request", "error", err)
				next.ServeHTTP(w, r)
				return
			}
			if count == 1 {
				rdb.Expire(context.Background(), key, window)
			}
			if count > limit {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"code":8,"message":"rate limit exceeded"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
