---
phase: 1
slug: project-scaffolding
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-11
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (standard library) |
| **Config file** | none — each service has its own go.mod |
| **Quick run command** | `go build ./...` per service |
| **Full suite command** | `cd auth && go build ./... && cd ../twofa && go build ./... && cd ../mpc && go build ./...` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go build ./...` in affected service
- **After every plan wave:** Run full suite command across all services
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 1-01-01 | 01 | 1 | INFRA-10 | — | N/A | build | `cd auth && go build ./...` | ❌ W0 | ⬜ pending |
| 1-01-02 | 01 | 1 | INFRA-09 | — | N/A | build | `cd auth && ./scripts/generate.sh && go build ./...` | ❌ W0 | ⬜ pending |
| 1-01-03 | 01 | 1 | INFRA-08 | — | N/A | build | `cd auth && go build ./...` | ❌ W0 | ⬜ pending |
| 1-02-01 | 02 | 1 | INFRA-01, INFRA-02 | — | N/A | build | `cd auth && go build ./...` | ❌ W0 | ⬜ pending |
| 1-02-02 | 02 | 1 | INFRA-11 | — | N/A | integration | `docker-compose up -d && docker-compose ps` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Go modules initialized (`go mod init`) for each service
- [ ] Proto tooling installed (protoc, protoc-gen-go, protoc-gen-go-grpc)

*Existing infrastructure covers test framework needs (go test is built-in).*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Docker Compose starts PG+Redis | INFRA-11 | Requires Docker daemon running | `docker-compose up -d && docker-compose ps` — verify services healthy |
| gRPC port listening | INFRA-01 | Requires running service process | Start service, `grpcurl -plaintext localhost:PORT list` |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
