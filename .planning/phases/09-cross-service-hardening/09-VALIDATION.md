---
phase: 9
slug: cross-service-hardening
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-12
---

# Phase 9 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (standard library) + minimock |
| **Config file** | none — go test built-in |
| **Quick run command** | `cd <service> && go test ./internal/... -count=1` |
| **Full suite command** | `cd auth && go test ./... -count=1 && cd ../twofa && go test ./... -count=1 && cd ../mpc && go test ./... -count=1` |
| **Estimated runtime** | ~15 seconds (all 3 services) |

---

## Sampling Rate

- **After every task commit:** Run quick run command for affected service
- **After every plan wave:** Run full suite command (all 3 services)
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 09-01-01 | 01 | 1 | INFRA-05 | — | Prometheus metrics interceptor records count + duration | unit | `cd auth && go test ./internal/middleware/ -run TestMetrics -count=1` | ❌ W0 | ⬜ pending |
| 09-01-02 | 01 | 1 | INFRA-07 | — | EventProducer publishes audit events, fire-and-forget | unit | `cd auth && go test ./internal/services/authService/ -run TestAudit -count=1` | ❌ W0 | ⬜ pending |
| 09-01-03 | 01 | 1 | INFRA-06 | — | slog JSON handler configured, no secrets logged | grep | `grep -rn 'password\|secret\|share_data\|encryption_key' auth/ twofa/ mpc/ --include='*.go' \| grep -i slog` | N/A | ⬜ pending |
| 09-01-04 | 01 | 1 | INFRA-04 | — | Ordered shutdown: gRPC → Kafka → Redis → PG | manual | Observe log output on SIGTERM | N/A | ⬜ pending |
| 09-02-01 | 02 | 1 | INFRA-07 | — | Auth audit events published for all operations | unit | `cd auth && go test ./internal/services/authService/ -run TestAudit -count=1` | ❌ W0 | ⬜ pending |
| 09-02-02 | 02 | 1 | INFRA-07 | — | TwoFA audit events published for all operations | unit | `cd twofa && go test ./internal/services/twofaService/ -run TestAudit -count=1` | ❌ W0 | ⬜ pending |
| 09-02-03 | 02 | 1 | INFRA-07 | — | MPC audit events published for all operations | unit | `cd mpc && go test ./internal/services/mpcService/ -run TestAudit -count=1` | ❌ W0 | ⬜ pending |
| 09-02-04 | 02 | 1 | SEC-02 | — | gRPC errors contain no internal state | grep | `grep -rn 'status.Error' auth/ twofa/ mpc/ --include='*.go'` | N/A | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `auth/internal/services/authService/mocks/event_producer_mock.go` — minimock generated
- [ ] `twofa/internal/services/twofaService/mocks/event_producer_mock.go` — minimock generated
- [ ] `mpc/internal/services/mpcService/mocks/event_producer_mock.go` — minimock generated
- [ ] Update all existing tests to pass NoOpProducer/mock as EventProducer parameter
- [ ] Install prometheus + kafka-go deps in all 3 go.mod files

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Ordered shutdown sequence | INFRA-04 | Requires running service + SIGTERM signal | Start service, send SIGTERM, verify log output shows correct order |
| Health check responds SERVING | INFRA-03 | Already implemented, smoke test only | `grpcurl -plaintext localhost:9090 grpc.health.v1.Health/Check` |
| Metrics endpoint accessible | INFRA-05 | Integration test requiring running service | `curl localhost:9100/metrics \| grep auth_requests_total` |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
