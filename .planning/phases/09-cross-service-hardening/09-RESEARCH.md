# Phase 9: Cross-Service Hardening - Research

**Researched:** 2026-04-12
**Domain:** Go microservice production-readiness (Prometheus, Kafka, slog, graceful shutdown)
**Confidence:** HIGH

## Summary

Phase 9 adds production-readiness infrastructure to all 3 services (auth, twofa, mpc): Prometheus metrics via gRPC interceptor + separate HTTP metrics endpoint, Kafka audit event publishing, structured slog JSON handler configuration, and ordered graceful shutdown. All 3 services already have gRPC health checks, basic GracefulStop, slog usage, and sanitized error responses from prior phases -- this phase upgrades and completes them.

The codebase is well-structured for these additions. Each service follows the same Clean Architecture pattern with bootstrap factories, middleware interceptors, and config.yaml. The work is highly repetitive across services (same pattern, 3 implementations) which makes it suitable for 2 plans: Plan 1 for shared infrastructure (metrics interceptor, Kafka producer, config changes, shutdown), Plan 2 for service-specific audit events and verification.

**Primary recommendation:** Implement a shared metrics interceptor pattern and Kafka EventProducer interface, then replicate across all 3 services. Use `promauto` for metric registration and `kafka.Writer` with fire-and-forget semantics.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Metrics collected via gRPC unary interceptor (request count + duration histogram per method). Added to existing chain in each service's bootstrap.
- **D-02:** Service-specific counters: `auth_requests_total{method, status}`, `auth_request_duration_seconds`, `twofa_operations_total{operation, status}`, `twofa_mpc_latency_seconds{node_id}`, `mpc_operations_total{node_id, operation, status}`.
- **D-03:** Separate HTTP listener per service for `/metrics` (auth :9100, twofa :9101, mpc :9102 defaults).
- **D-04:** Metrics port in config.yaml under `server.metrics_port`.
- **D-05:** Use `promauto` for metric registration. Package-level vars in dedicated `metrics.go` file.
- **D-06:** One Kafka topic per service: `auth.events`, `twofa.events`, `mpc.events`. Event key is `user_id`.
- **D-07:** Event schema: `{"user_id": string, "operation": string, "timestamp": ISO8601, "status": string, "node_id": string (mpc only)}`. NEVER secret data.
- **D-08:** Producer via `kafka-go` (`github.com/segmentio/kafka-go`). Writer with topic, LeastBytes balancer, async writes. Injected via interface.
- **D-09:** Fire-and-forget delivery. Failures logged via slog.Warn, never block main operation.
- **D-10:** Producer interface: `EventProducer` with `PublishEvent(ctx, event)` and `Close()`. Mock via minimock.
- **D-11:** Kafka configured in existing config.yaml `kafka` section (brokers already present). Add `topic` field.
- **D-12:** Audit events: Auth (user.registered, user.logged_in, user.logged_out, token.refreshed, token.refresh_reuse_detected), TwoFA (2fa.setup, 2fa.verified, 2fa.disabled, 2fa.status_checked), MPC (share.stored, share.retrieved, share.deleted).
- **D-13:** Replace default slog handler with `slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: configuredLevel})` at startup.
- **D-14:** Log level configurable via `server.log_level` in config.yaml. Default: info.
- **D-15:** `slog.SetDefault(logger)` in main.go -- no changes needed in existing service files.
- **D-16:** Verify no secret data in logs via grep audit.
- **D-17:** Ordered teardown: gRPC GracefulStop -> Kafka Close -> Redis Close -> PG Close -> Metrics HTTP Shutdown.
- **D-18:** 30-second timeout for shutdown via `context.WithTimeout`.
- **D-19:** Bootstrap returns Closer/cleanup function aggregated in order.
- **D-20:** Error sanitization audit -- verification pass, not rewrite.

### Claude's Discretion
- Exact interceptor chain ordering per service
- Prometheus metric bucket sizes for duration histograms
- Kafka Writer configuration details (batch size, batch timeout, max attempts)
- Whether to use shared audit event package or per-service event definitions
- slog attribute naming conventions
- Whether metrics interceptor wraps logging interceptor or vice versa
- Exact shutdown log messages

### Deferred Ideas (OUT OF SCOPE)
- Prometheus + Grafana dashboard configuration (MON-01 v2)
- Alerting rules (MON-02 v2)
- Stream interceptors (no streaming RPCs used)
- mTLS between services (ASEC-01 v2)
- Log aggregation (ELK/Loki)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| INFRA-03 | gRPC Health Check Protocol in each service | Already implemented in all 3 services (verified in bootstrap files). Phase 9 is a verification pass only. |
| INFRA-04 | Graceful shutdown with ordered teardown (gRPC -> Kafka -> Redis -> PG) | Currently only `GracefulStop()` exists. Research provides shutdown pattern with 30s timeout and ordered resource cleanup. |
| INFRA-05 | Prometheus metrics per service (requests total, duration, service-specific counters) | Research provides `promauto` pattern, metrics interceptor code, separate HTTP `/metrics` endpoint pattern. |
| INFRA-06 | Structured logging with slog -- secrets NEVER logged | slog already used everywhere. Research provides JSON handler config and grep-based secret audit pattern. |
| INFRA-07 | Kafka audit events per service (user_id, operation, timestamp -- no secret data) | Research provides kafka.Writer configuration, EventProducer interface, fire-and-forget pattern, AuditEvent struct. |
| SEC-02 | gRPC errors sanitized -- no internal state leaked | Already implemented well across all services (verified via grep). Phase 9 is a verification/audit pass. |
</phase_requirements>

## Standard Stack

### Core (New Dependencies)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/prometheus/client_golang` | v1.23.2 | Prometheus metrics collection and HTTP handler | De facto Go Prometheus client; `promauto` simplifies registration [VERIFIED: go list -m] |
| `github.com/segmentio/kafka-go` | v0.4.50 | Kafka producer for audit events | Already in CLAUDE.md spec; pure Go, no CGo dependency [VERIFIED: go list -m] |

### Already Present (No Changes)
| Library | Version | Purpose |
|---------|---------|---------|
| `log/slog` | stdlib | Structured logging (Go 1.21+) |
| `google.golang.org/grpc` | v1.80.0 | gRPC server + health checks |
| `gopkg.in/yaml.v3` | v3.0.1 | Config file parsing |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `promauto` | Manual `prometheus.MustRegister()` | promauto is simpler, auto-registers; manual gives explicit control. promauto recommended per D-05. |
| Per-service event definitions | Shared `pkg/audit` package | Shared package adds cross-module dependency in a multi-module repo. Per-service is simpler -- recommend per-service since services are separate Go modules. |

**Installation (per service):**
```bash
cd auth && go get github.com/prometheus/client_golang@v1.23.2 github.com/segmentio/kafka-go@v0.4.50
cd twofa && go get github.com/prometheus/client_golang@v1.23.2 github.com/segmentio/kafka-go@v0.4.50
cd mpc && go get github.com/prometheus/client_golang@v1.23.2 github.com/segmentio/kafka-go@v0.4.50
```

## Architecture Patterns

### Files to Add/Modify Per Service

```
<service>/
├── cmd/app/main.go                          # MODIFY: slog JSON handler, ordered shutdown, metrics HTTP server
├── config/config.go                         # MODIFY: add MetricsPort, LogLevel fields to ServerConfig
├── config.yaml                              # MODIFY: add server.metrics_port, server.log_level, update kafka.topic
├── internal/
│   ├── bootstrap/
│   │   ├── server.go (auth) / bootstrap.go  # MODIFY: add metrics interceptor to chain, create metrics HTTP server
│   │   └── kafka.go                         # NEW: Kafka producer factory (NewKafkaProducer)
│   ├── middleware/
│   │   ├── interceptors.go                  # MODIFY: add MetricsInterceptor function
│   │   └── metrics.go                       # NEW: promauto metric variables (package-level vars)
│   └── services/<serviceName>/
│       ├── <service>_service.go             # MODIFY: add EventProducer field + constructor param
│       ├── audit.go                         # NEW: AuditEvent struct, EventProducer interface, publish helpers
│       └── mocks/event_producer_mock.go     # NEW: minimock-generated mock
├── go.mod                                   # MODIFY: add prometheus + kafka-go deps
└── go.sum                                   # AUTO: updated by go get
```

### Pattern 1: Metrics Interceptor

**What:** Single gRPC unary interceptor that records both request count and duration histogram.
**When to use:** Every service's gRPC server.

```go
// Source: prometheus/client_golang promauto pattern [ASSUMED - standard Go Prometheus pattern]
// File: <service>/internal/middleware/metrics.go
package middleware

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    grpcRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "auth_requests_total", // service-specific prefix
            Help: "Total number of gRPC requests",
        },
        []string{"method", "status"},
    )
    grpcRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "auth_request_duration_seconds",
            Help:    "Duration of gRPC requests in seconds",
            Buckets: prometheus.DefBuckets, // 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10
        },
        []string{"method"},
    )
)
```

```go
// File: <service>/internal/middleware/interceptors.go (add function)
func MetricsInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
    start := time.Now()
    resp, err := handler(ctx, req)
    duration := time.Since(start).Seconds()

    st, _ := status.FromError(err)
    grpcRequestsTotal.WithLabelValues(info.FullMethod, st.Code().String()).Inc()
    grpcRequestDuration.WithLabelValues(info.FullMethod).Observe(duration)

    return resp, err
}
```

### Pattern 2: Interceptor Chain Ordering

**What:** Order interceptors so metrics wraps everything (outermost), then logging, then auth (innermost for MPC).
**Recommendation:** Metrics first (outermost), then logging, then auth. This means metrics captures the full duration including auth check time.

```go
// Auth service (no auth interceptor):
grpc.ChainUnaryInterceptor(
    middleware.MetricsInterceptor,  // outermost: captures total duration
    middleware.LoggingInterceptor,  // logs after handler returns
)

// MPC service (has auth interceptor):
grpc.ChainUnaryInterceptor(
    middleware.MetricsInterceptor,          // outermost
    middleware.LoggingInterceptor,          // logging
    middleware.AuthInterceptor(cfg.SharedSecret), // innermost: rejects unauthenticated before handler
)
```

**Note:** Auth and twofa currently use `grpc.UnaryInterceptor()` (single interceptor). Both MUST switch to `grpc.ChainUnaryInterceptor()` to add the metrics interceptor. [VERIFIED: codebase grep]

### Pattern 3: Kafka EventProducer Interface

**What:** Interface for audit event publishing, injectable into service layer.
**Recommendation:** Define per-service (not shared package) since services are separate Go modules.

```go
// File: <service>/internal/services/<serviceName>/audit.go
package <serviceName>

import (
    "context"
    "time"
)

//go:generate minimock -i EventProducer -o ./mocks/ -s _mock.go

// EventProducer publishes audit events to Kafka.
type EventProducer interface {
    PublishEvent(ctx context.Context, event AuditEvent) error
    Close() error
}

// AuditEvent represents a single audit log entry.
type AuditEvent struct {
    UserID    string `json:"user_id"`
    Operation string `json:"operation"`
    Timestamp string `json:"timestamp"` // ISO 8601
    Status    string `json:"status"`
    NodeID    string `json:"node_id,omitempty"` // MPC only
}

// NewAuditEvent creates an AuditEvent with current timestamp.
func NewAuditEvent(userID, operation, status string) AuditEvent {
    return AuditEvent{
        UserID:    userID,
        Operation: operation,
        Timestamp: time.Now().UTC().Format(time.RFC3339),
        Status:    status,
    }
}
```

### Pattern 4: Kafka Writer (Concrete Producer)

**What:** Concrete EventProducer implementation wrapping kafka.Writer.

```go
// File: <service>/internal/bootstrap/kafka.go
package bootstrap

import (
    "context"
    "encoding/json"
    "log/slog"
    "time"

    "github.com/segmentio/kafka-go"
    "<module>/internal/services/<serviceName>"
)

// KafkaProducer implements EventProducer using kafka-go Writer.
type KafkaProducer struct {
    writer *kafka.Writer
}

// NewKafkaProducer creates a Kafka producer. Returns no-op producer if brokers empty.
func NewKafkaProducer(brokers []string, topic string) <serviceName>.EventProducer {
    if len(brokers) == 0 || brokers[0] == "" {
        slog.Warn("Kafka not configured, audit events disabled")
        return &NoOpProducer{}
    }
    return &KafkaProducer{
        writer: &kafka.Writer{
            Addr:         kafka.TCP(brokers...),
            Topic:        topic,
            Balancer:     &kafka.LeastBytes{},
            BatchSize:    100,
            BatchTimeout: 10 * time.Millisecond,
            Async:        true,
        },
    }
}

func (p *KafkaProducer) PublishEvent(ctx context.Context, event <serviceName>.AuditEvent) error {
    data, err := json.Marshal(event)
    if err != nil {
        return err
    }
    return p.writer.WriteMessages(ctx, kafka.Message{
        Key:   []byte(event.UserID),
        Value: data,
    })
}

func (p *KafkaProducer) Close() error {
    return p.writer.Close()
}

// NoOpProducer discards events when Kafka is unavailable.
type NoOpProducer struct{}

func (p *NoOpProducer) PublishEvent(_ context.Context, _ <serviceName>.AuditEvent) error { return nil }
func (p *NoOpProducer) Close() error { return nil }
```

### Pattern 5: Fire-and-Forget Audit Publishing in Service Layer

**What:** Service methods publish audit events after successful operations without blocking.

```go
// Example: auth/internal/services/authService/register.go (add at end of Register method)
func (s *AuthService) Register(ctx context.Context, email, password string) (*domain.TokenPair, error) {
    // ... existing logic ...
    
    // Fire-and-forget audit event
    if err := s.eventProducer.PublishEvent(ctx, NewAuditEvent(user.ID, "user.registered", "success")); err != nil {
        slog.Warn("failed to publish audit event", "operation", "user.registered", "user_id", user.ID, "error", err)
    }
    
    return tokenPair, nil
}
```

### Pattern 6: Ordered Graceful Shutdown

**What:** Shutdown resources in correct order with 30s hard deadline.

```go
// File: <service>/cmd/app/main.go (shutdown section)
<-sigCh
slog.Info("shutting down service")

shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
defer shutdownCancel()

// 1. Stop accepting new gRPC requests
grpcServer.GracefulStop()
slog.Info("gRPC server stopped")

// 2. Flush Kafka (pending audit events)
if err := kafkaProducer.Close(); err != nil {
    slog.Error("failed to close Kafka producer", "error", err)
}
slog.Info("Kafka producer closed")

// 3. Close Redis
if redisStorage != nil {
    redisStorage.Close()
    slog.Info("Redis connection closed")
}

// 4. Close PostgreSQL
pgStorage.Close()
slog.Info("PostgreSQL connection closed")

// 5. Shutdown metrics HTTP server
if err := metricsServer.Shutdown(shutdownCtx); err != nil {
    slog.Error("failed to shutdown metrics server", "error", err)
}
slog.Info("metrics server stopped")

cancel()
```

### Pattern 7: slog JSON Handler Configuration

**What:** One-line change in main.go to switch all slog output to structured JSON.

```go
// At the very start of main(), before any slog calls:
logLevel := slog.LevelInfo // parse from config
switch cfg.Server.LogLevel {
case "debug":
    logLevel = slog.LevelDebug
case "warn":
    logLevel = slog.LevelWarn
case "error":
    logLevel = slog.LevelError
}
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
slog.SetDefault(logger)
```

### Pattern 8: Metrics HTTP Server

**What:** Separate HTTP server for Prometheus scraping.

```go
import "github.com/prometheus/client_golang/prometheus/promhttp"

metricsServer := &http.Server{
    Addr:    fmt.Sprintf(":%d", cfg.Server.MetricsPort),
    Handler: promhttp.Handler(),
}
go func() {
    slog.Info("metrics server started", "port", cfg.Server.MetricsPort)
    if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        slog.Error("metrics server error", "error", err)
    }
}()
```

### Anti-Patterns to Avoid
- **Mixing metrics HTTP with gRPC port:** Prometheus needs HTTP, gRPC is a different protocol. Always use a separate listener.
- **Blocking on audit event publish:** Kafka failures must NEVER block the main operation. Always log and continue.
- **Using `grpc.UnaryInterceptor` with multiple interceptors:** Will panic at runtime. Use `grpc.ChainUnaryInterceptor` when >1 interceptor.
- **Registering duplicate Prometheus metrics:** Will panic. Use `promauto` which handles this, or guard with `prometheus.MustRegister`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Prometheus metric types | Custom counter/histogram structs | `promauto.NewCounterVec`, `promauto.NewHistogramVec` | Thread-safe, atomic operations, proper serialization |
| Metrics HTTP endpoint | Custom /metrics handler | `promhttp.Handler()` | Proper exposition format, Content-Type negotiation |
| Kafka serialization | Custom wire format | `encoding/json` + `kafka.Message` | Standard JSON, partition key support built-in |
| gRPC status code extraction | String parsing of error | `status.FromError(err)` | Handles nil errors, unwraps correctly |

## Common Pitfalls

### Pitfall 1: grpc.UnaryInterceptor vs ChainUnaryInterceptor
**What goes wrong:** Using `grpc.UnaryInterceptor()` with multiple interceptors causes runtime panic ("grpc: UnaryInterceptor called multiple times").
**Why it happens:** Auth and twofa currently use `grpc.UnaryInterceptor()` for their single logging interceptor.
**How to avoid:** Switch all services to `grpc.ChainUnaryInterceptor()` when adding metrics interceptor.
**Warning signs:** Panic on service startup. [VERIFIED: auth/internal/bootstrap/server.go line 16, twofa/internal/bootstrap/bootstrap.go line 103]

### Pitfall 2: Prometheus Metric Name Collisions
**What goes wrong:** Registering the same metric name twice panics.
**Why it happens:** If metrics.go is imported by multiple test packages or init() runs twice.
**How to avoid:** Use `promauto` which uses `prometheus.DefaultRegisterer`. Package-level vars are initialized once. In tests, use a fresh registry if needed.
**Warning signs:** Panic with "duplicate metrics collector registration attempted".

### Pitfall 3: kafka.Writer Async Mode and Close
**What goes wrong:** Messages lost if `Close()` is not called on async writer.
**Why it happens:** Async writer buffers messages. Abrupt process exit loses the buffer.
**How to avoid:** Always call `writer.Close()` during shutdown (step 2, before Redis/PG close). The `Close()` method flushes pending messages.
**Warning signs:** Missing audit events at shutdown.

### Pitfall 4: slog JSON Handler Must Be Set Before First Log
**What goes wrong:** Early log messages (before handler set) use default text format.
**Why it happens:** Config loading happens before slog setup, and config loading may log errors.
**How to avoid:** Set a basic JSON handler first with default level, then reconfigure after config is loaded. Or accept that config load errors are in text format (acceptable since they are fatal errors).
**Warning signs:** Mixed text/JSON in log output.

### Pitfall 5: Config Struct Changes Break Existing Tests
**What goes wrong:** Adding MetricsPort and LogLevel fields to ServerConfig may break config tests if they assert on exact struct.
**Why it happens:** config_test.go files exist for all 3 services.
**How to avoid:** Check existing config tests and update them to include new fields.
**Warning signs:** Test failures in config package after struct changes. [VERIFIED: config_test.go exists in all 3 services]

### Pitfall 6: Defer Order in main.go
**What goes wrong:** Go defers execute LIFO. If relying on defer for shutdown, order is reversed from what's needed.
**Why it happens:** Current auth/mpc use `defer pgStorage.Close()` which executes in wrong order.
**How to avoid:** Don't use defer for ordered shutdown. Handle all cleanup explicitly in the shutdown goroutine. Remove existing defers and move to explicit ordered shutdown.
**Warning signs:** Resource leaks if PG closes before Kafka flush.

### Pitfall 7: Service Constructor Changes Break Existing Tests
**What goes wrong:** Adding `EventProducer` parameter to `NewAuthService`, `NewTwoFAService`, `NewMPCService` breaks all existing tests.
**Why it happens:** Every test that creates a service instance now needs the new parameter.
**How to avoid:** When adding EventProducer to constructors, also update all test files to pass a mock or no-op producer. Generate mocks with minimock.
**Warning signs:** Compilation errors in `_test.go` files.

## Code Examples

### Config Struct Extension (auth example)

```go
// Source: existing config/config.go pattern [VERIFIED: codebase]
type ServerConfig struct {
    Port        int    `yaml:"port"`
    MetricsPort int    `yaml:"metrics_port"`
    LogLevel    string `yaml:"log_level"`
}
```

### Config YAML Extension (auth example)

```yaml
server:
  port: 9090
  metrics_port: 9100
  log_level: "info"
```

### Service Constructor with EventProducer (auth example)

```go
// Minimal change to auth_service.go
type AuthService struct {
    storage         Storage
    sessionStorage  SessionStorage
    eventProducer   EventProducer  // NEW
    privateKey      *rsa.PrivateKey
    publicKey       *rsa.PublicKey
    accessTokenTTL  time.Duration
    refreshTokenTTL time.Duration
}

func NewAuthService(
    storage Storage,
    sessionStorage SessionStorage,
    eventProducer EventProducer,  // NEW parameter
    privateKey *rsa.PrivateKey,
    publicKey *rsa.PublicKey,
    accessTokenTTL time.Duration,
    refreshTokenTTL time.Duration,
) *AuthService {
    return &AuthService{
        storage:         storage,
        sessionStorage:  sessionStorage,
        eventProducer:   eventProducer,
        privateKey:      privateKey,
        publicKey:       publicKey,
        accessTokenTTL:  accessTokenTTL,
        refreshTokenTTL: refreshTokenTTL,
    }
}
```

## Discretion Recommendations

### Interceptor Chain Ordering
**Recommendation:** MetricsInterceptor (outermost) -> LoggingInterceptor -> AuthInterceptor (innermost, MPC only). Rationale: metrics captures total request time including auth overhead; logging logs after handler returns; auth rejects before handler runs. [ASSUMED]

### Prometheus Histogram Buckets
**Recommendation:** Use `prometheus.DefBuckets` (0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10 seconds). These are standard and appropriate for gRPC services. Twofa may have higher latency due to MPC calls, but DefBuckets covers up to 10s which is sufficient. [ASSUMED]

### Kafka Writer Configuration
**Recommendation:** `BatchSize: 100`, `BatchTimeout: 10ms`, `MaxAttempts: 3`, `Async: true`. This balances throughput (batching) with latency (short timeout) for audit events. Since fire-and-forget, async mode is appropriate. [ASSUMED]

### Shared vs Per-Service Event Definitions
**Recommendation:** Per-service event definitions. Services are separate Go modules -- a shared package would require a new Go module or import paths between modules. Per-service is simpler and follows existing patterns where each service defines its own interfaces. The AuditEvent struct is trivial (5 fields) -- duplication is acceptable. [VERIFIED: separate go.mod per service]

### slog Attribute Naming
**Recommendation:** Use snake_case consistent with existing slog calls: `"method"`, `"duration"`, `"error"`, `"user_id"`, `"operation"`, `"node_id"`. Already established in codebase. [VERIFIED: codebase grep shows snake_case usage]

### Shutdown Log Messages
**Recommendation:** Consistent format: `"shutting down <service>"`, `"gRPC server stopped"`, `"Kafka producer closed"`, `"Redis connection closed"`, `"PostgreSQL connection closed"`, `"metrics server stopped"`, `"<service> shutdown complete"`. [ASSUMED]

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `grpc.UnaryInterceptor` (single) | `grpc.ChainUnaryInterceptor` (multiple) | gRPC v1.28+ | Must switch auth and twofa for multiple interceptors |
| Manual prometheus.Register | `promauto` auto-registration | client_golang v1.0+ | Simpler metric declaration |
| `log` package | `log/slog` (stdlib) | Go 1.21 (2023) | Already adopted; JSON handler config is the upgrade |
| kafka-go `NewWriter` | Direct `&kafka.Writer{}` struct | kafka-go v0.4+ | Both work; struct literal preferred for clarity |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | MetricsInterceptor -> LoggingInterceptor -> AuthInterceptor ordering is optimal | Discretion Recommendations | Metrics might not capture auth rejection latency correctly if ordering is reversed; LOW risk |
| A2 | prometheus.DefBuckets adequate for all services | Discretion Recommendations | TwoFA MPC latency may need custom buckets if calls routinely >10s; LOW risk |
| A3 | kafka.Writer BatchSize=100, BatchTimeout=10ms is appropriate | Discretion Recommendations | Suboptimal batching in high/low throughput scenarios; LOW risk since fire-and-forget |
| A4 | Shutdown log message format | Discretion Recommendations | No functional impact; cosmetic only |

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + gotest.tools/v3 + minimock |
| Config file | None (Go convention: `go test ./...`) |
| Quick run command | `go test ./internal/...` (per service) |
| Full suite command | `cd auth && go test ./... && cd ../twofa && go test ./... && cd ../mpc && go test ./...` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| INFRA-03 | gRPC health check responds SERVING | manual-smoke | `grpcurl -plaintext localhost:9090 grpc.health.v1.Health/Check` | N/A (already implemented, verify only) |
| INFRA-04 | Ordered shutdown: gRPC -> Kafka -> Redis -> PG | manual-smoke | Observe log output on SIGTERM | N/A (integration test, manual) |
| INFRA-05 | Prometheus metrics exposed on /metrics | manual-smoke | `curl localhost:9100/metrics \| grep auth_requests_total` | N/A (Wave 0: add interceptor unit test) |
| INFRA-06 | slog JSON output, no secrets logged | unit + grep | `grep -rn 'password\|secret\|share_data\|encryption_key' --include='*.go' \| grep slog` | N/A (grep audit, not test file) |
| INFRA-07 | Kafka audit events published | unit | `go test ./internal/services/... -run TestAudit` | Wave 0 |
| SEC-02 | gRPC errors contain no internal state | grep-audit | `grep -rn 'status.Error' --include='*.go'` | N/A (manual review) |

### Sampling Rate
- **Per task commit:** `go test ./internal/... -count=1` (per service being modified)
- **Per wave merge:** Full suite across all 3 services
- **Phase gate:** All services compile, tests pass, `/metrics` returns expected metrics

### Wave 0 Gaps
- [ ] `auth/internal/services/authService/mocks/event_producer_mock.go` -- minimock generated
- [ ] `twofa/internal/services/twofaService/mocks/event_producer_mock.go` -- minimock generated
- [ ] `mpc/internal/services/mpcService/mocks/event_producer_mock.go` -- minimock generated
- [ ] Update all existing tests to pass NoOpProducer/mock as EventProducer parameter
- [ ] Install prometheus + kafka-go deps in all 3 go.mod files

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | N/A (auth already implemented) |
| V3 Session Management | no | N/A (sessions already implemented) |
| V4 Access Control | no | N/A |
| V5 Input Validation | no | N/A (no new user input) |
| V6 Cryptography | no | N/A |
| V7 Error Handling & Logging | yes | slog JSON handler, no secrets in logs, sanitized gRPC errors |
| V10 Malicious Code | yes | Audit events never contain secret data |

### Known Threat Patterns for This Phase

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Secret leakage in logs | Information Disclosure | Grep audit for password/secret/share/key in slog calls; never log request bodies containing sensitive fields |
| Secret leakage in error messages | Information Disclosure | Return generic gRPC error messages; log detailed errors server-side only |
| Secret leakage in audit events | Information Disclosure | AuditEvent struct has only user_id, operation, timestamp, status; no field for sensitive data |
| Metrics endpoint exposure | Information Disclosure | Separate port for /metrics; in production, restrict access via network policy (out of scope for code) |

## Sources

### Primary (HIGH confidence)
- Codebase inspection: all 3 services' main.go, bootstrap, middleware, config, service files [VERIFIED]
- `go list -m` for prometheus client_golang v1.23.2 and kafka-go v0.4.50 [VERIFIED]
- go.mod files confirming separate Go modules per service [VERIFIED]

### Secondary (MEDIUM confidence)
- prometheus/client_golang promauto pattern [ASSUMED - standard Go Prometheus pattern, well-established]
- kafka-go Writer API (struct literal, Async mode, Close flush) [ASSUMED - standard kafka-go usage pattern]

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - versions verified via go list -m, libraries specified in CLAUDE.md
- Architecture: HIGH - patterns derived from existing codebase (bootstrap, middleware, config patterns verified)
- Pitfalls: HIGH - interceptor chain issue verified in codebase, defer ordering analyzed from existing main.go code

**Research date:** 2026-04-12
**Valid until:** 2026-05-12 (stable libraries, no breaking changes expected)
