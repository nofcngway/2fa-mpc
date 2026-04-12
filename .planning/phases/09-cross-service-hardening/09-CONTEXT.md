# Phase 9: Cross-Service Hardening - Context

**Gathered:** 2026-04-12
**Status:** Ready for planning

<domain>
## Phase Boundary

Add production-readiness infrastructure to all 3 services (auth, twofa, mpc): Prometheus metrics via gRPC interceptor + separate HTTP metrics endpoint, Kafka audit event publishing (producer lifecycle, event schema, fire-and-forget delivery), structured slog JSON handler configuration, and ordered graceful shutdown (gRPC stop -> Kafka flush -> Redis close -> PG close). gRPC health checks, basic graceful shutdown, slog usage, and error sanitization already exist from Phases 1-8 -- this phase upgrades and completes them.

</domain>

<decisions>
## Implementation Decisions

### Prometheus Metrics
- **D-01:** Metrics collected via a gRPC unary interceptor that records request count and duration histogram per method. Interceptor added to the existing chain in each service's bootstrap (after auth interceptor in MPC, after logging interceptor in auth/twofa).
- **D-02:** Service-specific counters added manually in the service layer where needed: `auth_requests_total{method, status}`, `auth_request_duration_seconds`, `twofa_operations_total{operation, status}`, `twofa_mpc_latency_seconds{node_id}`, `mpc_operations_total{node_id, operation, status}` — per CLAUDE.md spec.
- **D-03:** Each service exposes a separate HTTP listener on a configurable metrics port (default :9100 for auth, :9101 for twofa, :9102 for mpc) serving `/metrics` via `promhttp.Handler()`. This avoids mixing metrics HTTP with gRPC on the main port.
- **D-04:** Metrics port configured in config.yaml under `server.metrics_port`. Bootstrap creates the HTTP server, main.go manages its lifecycle.
- **D-05:** Use `promauto` for metric registration — auto-registers with default prometheus registry. Metric variables declared as package-level vars in a dedicated `metrics.go` file per service's middleware or interceptors package.

### Kafka Audit Events
- **D-06:** One Kafka topic per service: `auth.events`, `twofa.events`, `mpc.events`. Event key is `user_id` for partition affinity.
- **D-07:** Event schema: `{"user_id": string, "operation": string, "timestamp": ISO8601, "status": string, "node_id": string (mpc only)}`. NEVER include passwords, TOTP secrets, share data, or encryption keys.
- **D-08:** Producer created in bootstrap using `kafka-go` (`github.com/segmentio/kafka-go`). Writer configured with topic, balancer (LeastBytes), and async writes. Injected into service via interface.
- **D-09:** Fire-and-forget delivery — audit event publishing failures logged via slog.Warn but never block the main operation. Service methods call `producer.PublishEvent(ctx, event)` after the main operation succeeds. If publish fails, log and continue.
- **D-10:** Producer interface: `EventProducer` with `PublishEvent(ctx context.Context, event AuditEvent) error` and `Close() error`. Concrete implementation wraps `kafka.Writer`. Mock via minimock for tests.
- **D-11:** Kafka connection configured in config.yaml under existing `kafka` section (brokers already present from Phase 1). Add `topic` field per service.
- **D-12:** Audit events published for key operations:
  - Auth: `user.registered`, `user.logged_in`, `user.logged_out`, `token.refreshed`, `token.refresh_reuse_detected`
  - TwoFA: `2fa.setup`, `2fa.verified`, `2fa.disabled`, `2fa.status_checked`
  - MPC: `share.stored`, `share.retrieved`, `share.deleted`

### Structured Logging Configuration
- **D-13:** Replace default slog handler with `slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: configuredLevel})` at startup in main.go. All services output structured JSON logs.
- **D-14:** Log level configurable via config.yaml under `server.log_level` field. Valid values: debug, info, warn, error. Default: info.
- **D-15:** Add `slog.SetDefault(logger)` in main.go so all `slog.Info()` calls throughout the codebase automatically use the JSON handler. No changes needed in existing service/storage files.
- **D-16:** Verify no secret data in logs — grep all services for slog calls, ensure no password, secret, share, or encryption key values are logged. Add log sanitization test if needed.

### Graceful Shutdown
- **D-17:** Upgrade existing shutdown in each service's main.go to ordered teardown: 1) `grpcServer.GracefulStop()`, 2) Kafka producer `Close()` (flush pending), 3) Redis client `Close()`, 4) PostgreSQL pool `Close()`. 5) Metrics HTTP server `Shutdown()`.
- **D-18:** Wrap shutdown in `context.WithTimeout(ctx, 30*time.Second)` as a hard deadline. If ordered teardown exceeds 30s, force exit.
- **D-19:** Bootstrap returns a `Closer` or cleanup function that main.go calls during shutdown. Each bootstrap factory returns its closer, aggregated in order.

### Error Sanitization Audit
- **D-20:** Audit all gRPC error responses across all services. Ensure no internal state (stack traces, SQL errors, file paths) leaks in error messages. Existing sanitization is good — this is a verification pass, not a rewrite.

### Claude's Discretion
- Exact interceptor chain ordering per service
- Prometheus metric bucket sizes for duration histograms
- Kafka Writer configuration details (batch size, batch timeout, max attempts)
- Whether to use a shared audit event package or per-service event definitions
- slog attribute naming conventions
- Whether metrics interceptor wraps logging interceptor or vice versa
- Exact shutdown log messages

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Architecture & Structure
- `CLAUDE.md` — Full project spec, Prometheus metrics requirements, Kafka audit rules (no secrets), structured logging (slog), graceful shutdown, gRPC health check
- `workspace/01 - Architecture/Overview.md` — System architecture overview

### Requirements
- `.planning/REQUIREMENTS.md` — INFRA-03, INFRA-04, INFRA-05, INFRA-06, INFRA-07, SEC-02

### Service Documentation
- `workspace/02 - Services/Auth Service.md` — Auth service metrics, audit events spec
- `workspace/02 - Services/TwoFA Service.md` — TwoFA service metrics, audit events spec
- `workspace/02 - Services/MPC Node.md` — MPC Node metrics, audit events spec

### Existing Code (all services)
- `auth/cmd/app/main.go` — Current shutdown pattern (needs upgrade)
- `auth/internal/bootstrap/server.go` — gRPC server creation with health check + interceptors
- `auth/internal/middleware/interceptors.go` — Logging interceptor (add metrics interceptor here)
- `auth/config/config.go` — Config struct with Kafka section (unused)
- `twofa/cmd/app/main.go` — Current shutdown pattern (needs upgrade)
- `twofa/internal/bootstrap/bootstrap.go` — gRPC server creation with health check + interceptors
- `twofa/internal/middleware/interceptors.go` — Logging interceptor
- `twofa/config/config.go` — Config struct with Kafka section (unused)
- `mpc/cmd/app/main.go` — Current shutdown pattern (needs upgrade)
- `mpc/internal/bootstrap/bootstrap.go` — gRPC server creation with health check + auth interceptor
- `mpc/internal/middleware/interceptors.go` — Auth + Logging interceptors
- `mpc/config/config.go` — Config struct with Kafka section (unused)

### Prior Phase Patterns
- `.planning/phases/01-project-scaffolding/01-CONTEXT.md` — D-10: bootstrap creates deps, log warnings for unavailable optional deps
- `.planning/phases/06-mpc-node-service/06-CONTEXT.md` — Auth interceptor pattern, service testing with mocks

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `auth/internal/middleware/interceptors.go` — LoggingInterceptor pattern, extend with MetricsInterceptor
- `mpc/internal/middleware/interceptors.go` — AuthInterceptor + LoggingInterceptor chain pattern, reference for interceptor ordering
- All 3 services: health check registration in bootstrap (already complete)
- All 3 services: slog usage throughout (handler config upgrade only needed in main.go)
- All 3 services: config.yaml with Kafka section already defined (brokers configured, topic field needed)
- All 3 services: signal.Notify + GracefulStop pattern (needs extension for ordered teardown)

### Established Patterns
- Bootstrap creates dependencies, injects into services via constructors
- Interfaces defined in service files, implementations in storage packages
- minimock for mock generation in tests
- One middleware file per service: `internal/middleware/interceptors.go`
- Config loaded via `config/config.go` with yaml unmarshaling

### Integration Points
- Each service's `internal/bootstrap/` — add Kafka producer + metrics HTTP server creation
- Each service's `cmd/app/main.go` — add slog JSON handler setup + ordered shutdown
- Each service's `internal/middleware/interceptors.go` — add Prometheus metrics interceptor
- Each service's service layer — add `EventProducer` dependency + publish calls after operations

</code_context>

<specifics>
## Specific Ideas

- Metrics interceptor should record both request count and duration in a single interceptor (not two separate ones)
- Kafka producer should be optional — if Kafka is unavailable at startup, service starts with a no-op producer that logs warnings (per Phase 1 D-10)
- slog JSON handler is a one-line change in main.go that affects all existing log calls automatically
- Shutdown ordering is critical: gRPC first (stop accepting), then flush Kafka (audit events), then close caches (Redis), then close DB (PostgreSQL)

</specifics>

<deferred>
## Deferred Ideas

- **Prometheus + Grafana dashboard configuration** — MON-01 in v2 requirements, not in Phase 9 scope
- **Alerting rules** — MON-02 in v2 requirements
- **Stream interceptors** — no streaming RPCs currently used, add if needed later
- **mTLS between services** — ASEC-01 in v2 requirements, replace shared secret auth
- **Log aggregation** — external concern (ELK/Loki), out of scope

None — discussion stayed within phase scope

</deferred>

---

*Phase: 09-cross-service-hardening*
*Context gathered: 2026-04-12*
