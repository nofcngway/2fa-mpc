---
phase: 09-cross-service-hardening
plan: 01
subsystem: observability-reliability
tags: [prometheus, slog, graceful-shutdown, metrics, interceptors]
dependency_graph:
  requires: []
  provides: [prometheus-metrics, json-logging, ordered-shutdown]
  affects: [auth, twofa, mpc]
tech_stack:
  added: [prometheus/client_golang/promauto, promhttp]
  patterns: [ChainUnaryInterceptor, slog-JSONHandler, ordered-shutdown]
key_files:
  created:
    - auth/internal/middleware/metrics.go
    - twofa/internal/middleware/metrics.go
    - mpc/internal/middleware/metrics.go
  modified:
    - auth/config/config.go
    - auth/config.yaml
    - auth/internal/middleware/interceptors.go
    - auth/internal/bootstrap/server.go
    - auth/cmd/app/main.go
    - twofa/config/config.go
    - twofa/config.yaml
    - twofa/internal/middleware/interceptors.go
    - twofa/internal/bootstrap/bootstrap.go
    - twofa/cmd/app/main.go
    - mpc/config/config.go
    - mpc/config.yaml
    - mpc/internal/middleware/interceptors.go
    - mpc/internal/bootstrap/bootstrap.go
    - mpc/cmd/app/main.go
    - auth/go.mod
    - auth/go.sum
    - twofa/go.mod
    - twofa/go.sum
    - mpc/go.mod
    - mpc/go.sum
decisions:
  - "MetricsInterceptor placed outermost in chain to capture all request durations including auth"
  - "Explicit ordered shutdown replaces defer-based cleanup (LIFO would reverse correct order)"
  - "MPC metrics port 9102 avoids conflict with gRPC port 9100"
metrics:
  duration: 306s
  completed: "2026-04-12T16:53:22Z"
  tasks_completed: 2
  tasks_total: 2
  files_changed: 24
---

# Phase 09 Plan 01: Cross-Service Observability and Reliability Summary

Prometheus metrics with promauto counter/histogram, slog JSON handler with configurable log level, and ordered graceful shutdown (gRPC -> Kafka placeholder -> Redis -> MPC conns -> PG -> metrics HTTP) across all 3 services.

## What Was Done

### Task 1: Config extensions + metrics interceptor (8d02985)

- Extended `ServerConfig` with `MetricsPort` and `LogLevel` fields in auth, twofa, mpc
- Updated `config.yaml` in all 3 services with metrics_port and log_level values
- Created `metrics.go` in each service's middleware package with promauto-registered counter (`*_requests_total` / `*_operations_total`) and histogram (`*_request_duration_seconds` / `*_mpc_latency_seconds`)
- Added `MetricsInterceptor` function to all 3 `interceptors.go` files
- Switched auth and twofa from `grpc.UnaryInterceptor` to `grpc.ChainUnaryInterceptor`
- Added MetricsInterceptor as outermost interceptor in all 3 services
- Installed `prometheus/client_golang v1.23.2` dependency

### Task 2: slog JSON handler + metrics HTTP + ordered shutdown (a602fa7)

- Rewrote all 3 `main.go` files with:
  - `slog.NewJSONHandler(os.Stdout, ...)` with configurable log level from config
  - Prometheus HTTP metrics server on separate port via `promhttp.Handler()`
  - Ordered graceful shutdown with 30s timeout (no defers for resource cleanup)
- Shutdown order per service:
  - **auth**: gRPC stop -> Redis close -> PG close -> metrics HTTP shutdown
  - **twofa**: gRPC stop -> Redis close -> MPC connections close -> PG close -> metrics HTTP shutdown
  - **mpc**: gRPC stop -> PG close -> metrics HTTP shutdown
- Verified error sanitization: all `status.Error` calls use generic messages (no SQL, paths, stack traces)
- Verified log secret audit: zero matches for password/secret/share_data/encryption_key in slog calls

## Verification Results

- All 3 services compile: `go build ./...` passes
- Config tests pass: `go test ./config/...` in auth, twofa, mpc
- Error sanitization audit: all gRPC errors use generic messages
- Log secret audit: zero instances of secret data in slog calls

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Missing go.sum entries for prometheus transitive dependencies**
- **Found during:** Task 1, Step 8
- **Issue:** `go get` added prometheus to go.mod but missed transitive deps in go.sum
- **Fix:** Ran `go mod tidy` in all 3 services
- **Files modified:** auth/go.sum, twofa/go.sum, mpc/go.sum
- **Commit:** 8d02985

## Self-Check: PASSED

- All 3 metrics.go files exist
- Commit 8d02985 found (Task 1)
- Commit a602fa7 found (Task 2)
