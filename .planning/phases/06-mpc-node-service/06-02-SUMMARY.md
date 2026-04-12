---
phase: 06-mpc-node-service
plan: 02
subsystem: mpc-service
tags: [grpc-handlers, auth-interceptor, bootstrap, security, constant-time]
dependency_graph:
  requires: [06-01]
  provides: [mpc-grpc-api, mpc-auth-interceptor, mpc-bootstrap-validation]
  affects: []
tech_stack:
  added: []
  patterns: [chain-unary-interceptor, constant-time-compare, grpc-error-mapping]
key_files:
  created:
    - mpc/internal/middleware/interceptors_test.go
  modified:
    - mpc/internal/middleware/interceptors.go
    - mpc/internal/api/mpc_service_api/store_share.go
    - mpc/internal/api/mpc_service_api/retrieve_share.go
    - mpc/internal/api/mpc_service_api/delete_share.go
    - mpc/internal/bootstrap/bootstrap.go
    - mpc/cmd/app/main.go
decisions:
  - AuthInterceptor uses closure pattern to capture expected secret at creation time
  - Health check excluded by exact FullMethod string match for /grpc.health.v1.Health/Check
  - Bootstrap validates encryption key length at startup to fail fast on misconfiguration
metrics:
  duration: 142s
  completed: "2026-04-12T12:52:41Z"
  tasks_completed: 2
  tasks_total: 2
  test_count: 25
  files_changed: 7
---

# Phase 06 Plan 02: MPC Node gRPC Handlers + Auth Interceptor Summary

Shared-secret auth interceptor with constant-time comparison, three gRPC handler implementations, and bootstrap key validation with ChainUnaryInterceptor

## What Was Done

### Task 1: Auth interceptor with shared-secret validation + tests (TDD)

- Added `AuthInterceptor` to interceptors.go using `subtle.ConstantTimeCompare` for timing-safe secret validation
- Health check requests (`/grpc.health.v1.Health/Check`) bypass authentication
- Missing metadata, missing authorization key, empty value, and wrong secret all return `codes.Unauthenticated`
- 7 unit tests covering all authentication scenarios (valid, wrong, missing metadata, empty, missing key, health check, constant-time)

### Task 2: gRPC handlers + bootstrap update

- **StoreShare handler**: validates user_id, share_data, share_index; delegates to service; maps ErrDuplicateShare to AlreadyExists
- **RetrieveShare handler**: validates user_id, share_index; delegates to service; maps ErrShareNotFound to NotFound, other errors to Internal
- **DeleteShare handler**: validates user_id; idempotent deletion returns deleted_count (including 0 for no-op)
- **Bootstrap NewMPCService**: validates encryption key is exactly 32 bytes, returns error on mismatch
- **Bootstrap NewGRPCServer**: replaced `grpc.UnaryInterceptor` with `grpc.ChainUnaryInterceptor(AuthInterceptor, LoggingInterceptor)`
- **main.go**: updated to handle NewMPCService error return and pass cfg to NewGRPCServer

## Commits

| Task | Commit | Message |
|------|--------|---------|
| 1 | cd6eec2 | feat(06-02): auth interceptor with shared-secret validation + 7 tests |
| 2 | 5bffcd7 | feat(06-02): gRPC handlers + bootstrap key validation + ChainUnaryInterceptor |

## Test Results

```
25 tests, 0 failures
- 7 auth interceptor tests (valid secret, wrong secret, missing metadata, empty, missing key, health check, constant-time)
- 18 service layer tests (from Plan 01, all still passing)
```

## Deviations from Plan

None -- plan executed exactly as written.

## Known Stubs

None -- all three gRPC handlers are fully implemented with input validation, service delegation, and proper error mapping.

## Threat Surface Verification

All threat mitigations from the plan's threat model are implemented:
- T-06-06: AuthInterceptor validates shared secret with subtle.ConstantTimeCompare
- T-06-07: Error messages are generic ("missing authorization", "invalid authorization", "failed to store share")
- T-06-08: Health check excluded by exact FullMethod match; all other methods require auth
- T-06-09: Encryption key validated at startup (32 bytes); service refuses to start with invalid key
- T-06-10: gRPC handler errors use generic messages; no internal error details returned to client
- T-06-11: Input validation on all handlers: user_id non-empty, share_data non-empty, share_index >= 0

## Self-Check: PASSED

All 7 created/modified files verified on disk. Both commit hashes (cd6eec2, 5bffcd7) found in git log.
