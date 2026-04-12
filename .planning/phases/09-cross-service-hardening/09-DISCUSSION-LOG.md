# Phase 9: Cross-Service Hardening - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md -- this log preserves the alternatives considered.

**Date:** 2026-04-12
**Phase:** 09-cross-service-hardening
**Areas discussed:** Prometheus metrics design, Kafka audit events, Structured logging configuration, Graceful shutdown ordering
**Mode:** --auto (all recommended defaults selected)

---

## Prometheus Metrics Design

| Option | Description | Selected |
|--------|-------------|----------|
| gRPC interceptor + service counters | Interceptor for request count/duration, manual counters in service layer for domain metrics | ✓ |
| Manual instrumentation only | Add metrics calls directly in handlers/services, no interceptor | |
| Third-party middleware (go-grpc-prometheus) | Use pre-built interceptor library | |

**User's choice:** [auto] gRPC interceptor + service counters (recommended)
**Notes:** Separate HTTP listener per service for /metrics endpoint. promauto for registration.

| Option | Description | Selected |
|--------|-------------|----------|
| Separate HTTP listener for /metrics | Dedicated port (9100-9102) per service | ✓ |
| Serve on gRPC port via reflection | Use gRPC server reflection to expose metrics | |
| Embed in existing HTTP (gateway only) | Only expose metrics through gateway | |

**User's choice:** [auto] Separate HTTP listener (recommended)
**Notes:** Configurable via server.metrics_port in config.yaml

---

## Kafka Audit Events

| Option | Description | Selected |
|--------|-------------|----------|
| One topic per service | auth.events, twofa.events, mpc.events with user_id key | ✓ |
| Single shared topic | All services write to audit.events with service field | |
| Domain-based topics | Topics per operation type (auth.login, 2fa.verify, etc.) | |

**User's choice:** [auto] One topic per service (recommended)
**Notes:** None

| Option | Description | Selected |
|--------|-------------|----------|
| Fire-and-forget | Log failure, never block main operation | ✓ |
| At-least-once with retry | Retry failed publishes, still async | |
| Synchronous guaranteed | Block until Kafka confirms receipt | |

**User's choice:** [auto] Fire-and-forget (recommended)
**Notes:** Audit failure must not affect user-facing operations

| Option | Description | Selected |
|--------|-------------|----------|
| Bootstrap creates, inject via interface | Producer created in bootstrap, EventProducer interface, minimock for tests | ✓ |
| Lazy initialization | Create producer on first publish | |
| Global singleton | Package-level producer variable | |

**User's choice:** [auto] Bootstrap creates, inject via interface (recommended)
**Notes:** Matches existing DI pattern from Phases 1-8

---

## Structured Logging Configuration

| Option | Description | Selected |
|--------|-------------|----------|
| slog.NewJSONHandler + SetDefault | JSON handler in main.go, affects all existing slog calls | ✓ |
| Custom handler wrapper | Wrap JSON handler with additional fields (service_name, version) | |
| Keep default text handler | Current behavior, no change | |

**User's choice:** [auto] slog.NewJSONHandler + SetDefault (recommended)
**Notes:** One-line change in main.go, no modifications needed in existing files

| Option | Description | Selected |
|--------|-------------|----------|
| Config.yaml log_level field | server.log_level: info/debug/warn/error | ✓ |
| Environment variable | LOG_LEVEL env var | |
| Hardcoded INFO | No configuration | |

**User's choice:** [auto] Config.yaml log_level field (recommended)
**Notes:** Default: info

---

## Graceful Shutdown Ordering

| Option | Description | Selected |
|--------|-------------|----------|
| Ordered teardown with timeout | GracefulStop -> Kafka flush -> Redis close -> PG close, 30s deadline | ✓ |
| Parallel close all | Close everything simultaneously | |
| Current pattern (GracefulStop only) | Keep existing, no resource cleanup | |

**User's choice:** [auto] Ordered teardown with timeout (recommended)
**Notes:** Bootstrap returns closers, main.go calls in order

---

## Claude's Discretion

- Exact interceptor chain ordering per service
- Prometheus metric bucket sizes for duration histograms
- Kafka Writer configuration details (batch size, batch timeout, max attempts)
- Whether to use a shared audit event package or per-service event definitions
- slog attribute naming conventions
- Whether metrics interceptor wraps logging interceptor or vice versa
- Exact shutdown log messages

## Deferred Ideas

- Prometheus + Grafana dashboard configuration (MON-01, v2)
- Alerting rules (MON-02, v2)
- Stream interceptors (no streaming RPCs currently)
- mTLS between services (ASEC-01, v2)
- Log aggregation (external concern)
