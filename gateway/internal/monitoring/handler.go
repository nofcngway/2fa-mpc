package monitoring

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// SnapshotHandler returns an HTTP handler that responds with a fresh
// monitoring snapshot. The handler is intentionally trivial — auth/rate-limit
// middleware in front of it stays the source of truth for access control.
//
// Accept any authenticated user for now; gating to an admin role is a future
// concern (see TODO in the README) once user roles exist.
func SnapshotHandler(c *Collector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c == nil {
			http.Error(w, `{"code":12,"message":"monitoring is not configured"}`, http.StatusServiceUnavailable)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		snap, err := c.Build(ctx)
		if err != nil {
			slog.Warn("monitoring snapshot failed", "error", err)
			http.Error(w, `{"code":13,"message":"failed to build snapshot"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if err := json.NewEncoder(w).Encode(snap); err != nil {
			slog.Warn("monitoring snapshot encode failed", "error", err)
		}
	}
}
