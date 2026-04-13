---
phase: 10-refactoring-bootstrap-split-slog-logging-dependency-inversio
plan: 01
subsystem: twofa-bootstrap
tags: [refactoring, bootstrap, clean-architecture]
dependency_graph:
  requires: []
  provides: [twofa-bootstrap-split, mpc-clients-rename]
  affects: [twofa/internal/bootstrap/]
tech_stack:
  added: []
  patterns: [per-component-bootstrap-files]
key_files:
  created:
    - twofa/internal/bootstrap/pgstorage.go
    - twofa/internal/bootstrap/redisstorage.go
    - twofa/internal/bootstrap/service.go
    - twofa/internal/bootstrap/service_api.go
    - twofa/internal/bootstrap/server.go
  modified:
    - twofa/internal/bootstrap/mpc_clients.go (renamed from mpc_adapter.go, consolidated MPC functions)
  deleted:
    - twofa/internal/bootstrap/bootstrap.go
decisions:
  - Split TwoFA bootstrap into per-component files matching Auth pattern
  - Consolidated all MPC client code (adapter + factory + interceptor) into mpc_clients.go
metrics:
  duration: 147s
  completed: "2026-04-13T18:19:10Z"
  tasks_completed: 2
  tasks_total: 2
  files_changed: 7
---

# Phase 10 Plan 01: TwoFA Bootstrap Split Summary

Split TwoFA consolidated bootstrap.go into 7 per-component files matching Auth service pattern, renamed mpc_adapter.go to mpc_clients.go.

## Tasks Completed

| Task | Name | Commit | Key Changes |
|------|------|--------|-------------|
| 1 | Split TwoFA bootstrap.go into per-component files | f78784f | Created 5 new files (pgstorage, redisstorage, service, service_api, server), moved MPC functions into mpc_adapter.go, deleted bootstrap.go |
| 2 | Rename mpc_adapter.go to mpc_clients.go | 98fe73f | Pure file rename via git mv |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Moved NewMPCClients and authMetadataInterceptor before deleting bootstrap.go**
- **Found during:** Task 1
- **Issue:** bootstrap.go contained NewMPCClients and authMetadataInterceptor which were not in mpc_adapter.go. Deleting bootstrap.go without moving them first would break compilation.
- **Fix:** Consolidated all MPC client-related code (adapter struct + methods + NewMPCClients factory + authMetadataInterceptor) into mpc_adapter.go during Task 1, before deletion.
- **Files modified:** twofa/internal/bootstrap/mpc_adapter.go
- **Commit:** f78784f

## Final Bootstrap Directory

```
twofa/internal/bootstrap/
  kafka.go         - KafkaProducer + NoOpProducer (unchanged)
  mpc_clients.go   - mpcClientAdapter + NewMPCClients + authMetadataInterceptor
  pgstorage.go     - NewPGStorage factory
  redisstorage.go  - NewSessionStorage factory
  server.go        - NewGRPCServer factory
  service.go       - NewTwoFAService factory
  service_api.go   - NewTwoFAServiceAPI factory
```

## Verification

- `go build ./...` exits 0
- `go test ./...` all tests pass (crypto, shamir, totp, twofaService)
- bootstrap.go does not exist
- mpc_adapter.go does not exist
- All 7 expected files present

## Self-Check: PASSED

- All 6 created/renamed files: FOUND
- bootstrap.go: CONFIRMED DELETED
- mpc_adapter.go: CONFIRMED DELETED
- Commit f78784f: FOUND
- Commit 98fe73f: FOUND
