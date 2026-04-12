---
phase: 03-auth-sessions-jwt
plan: 03
subsystem: auth-grpc-handlers
tags: [grpc, handlers, auth, jwt, sessions]
dependency_graph:
  requires: [03-02]
  provides: [auth-grpc-handlers-wired]
  affects: [auth-service-api]
tech_stack:
  added: []
  patterns: [grpc-error-mapping, handler-delegation]
key_files:
  created:
    - auth/internal/api/auth_service_api/logout_all.go
  modified:
    - auth/internal/api/auth_service_api/login.go
    - auth/internal/api/auth_service_api/refresh_token.go
    - auth/internal/api/auth_service_api/validate_token.go
    - auth/internal/api/auth_service_api/logout.go
decisions:
  - Register handler already had tokens wired from Plan 02 -- no update needed
metrics:
  duration: 75s
  completed: 2026-04-12T04:44:17Z
  tasks_completed: 2
  tasks_total: 2
  files_changed: 5
requirements:
  - SEC-03
---

# Phase 03 Plan 03: gRPC Handler Wiring Summary

All six gRPC handler stubs replaced with real implementations delegating to AuthService, with proper gRPC error code mapping and SEC-03 compliance (password_hash never in responses).

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Implement Login, RefreshToken, ValidateToken handlers | 6375cb2 | login.go, refresh_token.go, validate_token.go |
| 2 | Implement Logout, LogoutAll handlers | ddbb53d | logout.go, logout_all.go |

## What Was Built

- **Login handler**: Validates email/password input, delegates to service.Login, returns TokenPair + User (no password_hash). Maps ErrInvalidCredentials to codes.Unauthenticated (generic message, no credential enumeration).
- **RefreshToken handler**: Validates refresh token input, delegates to service.RefreshToken, returns new TokenPair. Maps ErrInvalidToken and ErrTokenRevoked to codes.Unauthenticated.
- **ValidateToken handler**: Validates access token input, delegates to service.ValidateToken, returns user_id and email.
- **Logout handler**: Validates refresh token input, delegates to service.Logout, returns empty LogoutResponse.
- **LogoutAll handler** (new file): Validates user_id input, delegates to service.LogoutAll, returns empty LogoutAllResponse.
- **Register handler**: Already correctly implemented in Plan 02 with TokenPair population -- no changes needed.

## Security Compliance

- **SEC-03/D-16**: password_hash field never set in any proto User response -- verified with grep (zero matches)
- **T-03-09**: No internal state leaked in error messages -- all errors are generic
- **T-03-10**: Generic gRPC error messages only ("invalid credentials", "internal error")
- **T-03-11**: Login returns same Unauthenticated code for wrong email and wrong password

## Verification Results

- `go build ./...` -- passes
- `go vet ./...` -- passes
- `go test ./... -v -count=1` -- all 21 tests pass
- `grep -rn "PasswordHash" auth/internal/api/auth_service_api/` -- zero matches (SEC-03)
- `grep -rn "Unimplemented" auth/internal/api/auth_service_api/` -- only embedded struct reference

## Deviations from Plan

### Register handler already complete

**Found during:** Task 2
**Issue:** Plan specified updating register.go to populate TokenPair, but register.go was already correctly implemented in Plan 02 with full token support.
**Action:** No changes needed -- skipped register.go modification.
**Impact:** None -- handler already met all requirements.

## Known Stubs

None -- all handler stubs have been replaced with real implementations.
