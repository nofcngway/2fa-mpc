# Roadmap: MPC-2FA

## Milestones

- ✅ **v1.0 MPC-2FA** — Phases 1-13 (shipped 2026-04-16)

## Phases

<details>
<summary>✅ v1.0 MPC-2FA (Phases 1-13) — SHIPPED 2026-04-16</summary>

- [x] Phase 1: Project Scaffolding (6/6 plans) — Go modules, proto generation, Docker Compose, config, Clean Architecture skeleton
- [x] Phase 2: Auth Registration (2/2 plans) — User registration with password validation, bcrypt cost=12
- [x] Phase 3: Auth Sessions & JWT (3/3 plans) — Login, JWT RS256, refresh rotation, theft detection, logout
- [x] Phase 4: Shamir Secret Sharing (2/2 plans) — GF(256) arithmetic, split/combine in pure Go
- [x] Phase 5: TOTP Implementation (2/2 plans) — RFC 6238, provisioning URI, time window tests
- [x] Phase 6: MPC Node Service (2/2 plans) — AES-256-GCM encrypted share storage, gRPC auth interceptor
- [x] Phase 7: TwoFA Setup Flow (2/3 plans) — Secret generation, Shamir split, MPC distribution, backup codes, zeroization
- [x] Phase 8: TwoFA Verification & Management (2/2 plans) — OTP verify, rate limiting, disable, status
- [x] Phase 9: Cross-Service Hardening (2/2 plans) — Health checks, graceful shutdown, Prometheus, slog, Kafka audit
- [x] Phase 10: Bootstrap Refactoring (3/3 plans) — Per-component bootstrap files, slog extraction, DI fixes
- [x] Phase 11: Domain Rename (3/3 plans) — Rename models/ to domain/, consolidate error sentinels, update docs
- [x] Phase 12: Dockerfiles & Makefiles (4/4 plans) — Multi-stage Dockerfiles, root docker-compose, Makefiles with help
- [x] Phase 13: Kafka UI (1/1 plan) — Kafbat Kafka UI in docker-compose

</details>

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Project Scaffolding | v1.0 | 6/6 | Complete | 2026-04-11 |
| 2. Auth Registration | v1.0 | 2/2 | Complete | 2026-04-11 |
| 3. Auth Sessions & JWT | v1.0 | 3/3 | Complete | 2026-04-11 |
| 4. Shamir Secret Sharing | v1.0 | 2/2 | Complete | 2026-04-12 |
| 5. TOTP Implementation | v1.0 | 2/2 | Complete | 2026-04-12 |
| 6. MPC Node Service | v1.0 | 2/2 | Complete | 2026-04-12 |
| 7. TwoFA Setup Flow | v1.0 | 2/3 | Complete | 2026-04-12 |
| 8. TwoFA Verification & Mgmt | v1.0 | 2/2 | Complete | 2026-04-12 |
| 9. Cross-Service Hardening | v1.0 | 2/2 | Complete | 2026-04-12 |
| 10. Bootstrap Refactoring | v1.0 | 3/3 | Complete | 2026-04-13 |
| 11. Domain Rename | v1.0 | 3/3 | Complete | 2026-04-14 |
| 12. Dockerfiles & Makefiles | v1.0 | 4/4 | Complete | 2026-04-14 |
| 13. Kafka UI | v1.0 | 1/1 | Complete | 2026-04-14 |

## Backlog

### Phase 999.1: Follow-up — Phase 7 incomplete plans (BACKLOG)

**Goal:** Resolve plan that ran without producing a summary during Phase 7 execution
**Source phase:** 7
**Deferred at:** 2026-04-16 during /gsd-next advancement
**Plans:**
- [ ] 07-03: gap-closure (ran, no SUMMARY.md)
