---
phase: 03-auth-sessions-jwt
plan: 02
subsystem: auth-service-layer
tags: [auth, login, jwt, refresh-token, theft-detection, logout, session]
dependency_graph:
  requires: [03-01]
  provides: [Login, RefreshToken, ValidateToken, Logout, LogoutAll, Register-auto-login]
  affects: [auth-grpc-handlers, gateway-auth]
tech_stack:
  added: []
  patterns: [token-family-rotation, theft-detection, bcrypt-auth, tdd]
key_files:
  created:
    - auth/internal/services/authService/login.go
    - auth/internal/services/authService/login_test.go
    - auth/internal/services/authService/refresh_token.go
    - auth/internal/services/authService/refresh_token_test.go
    - auth/internal/services/authService/validate_token.go
    - auth/internal/services/authService/validate_token_test.go
    - auth/internal/services/authService/logout.go
    - auth/internal/services/authService/logout_test.go
    - auth/internal/services/authService/logout_all.go
    - auth/internal/services/authService/logout_all_test.go
  modified:
    - auth/internal/services/authService/register.go
    - auth/internal/services/authService/register_test.go
    - auth/internal/api/auth_service_api/register.go
decisions:
  - Login returns same ErrInvalidCredentials for wrong email and wrong password to prevent credential enumeration (T-03-06)
  - RefreshToken uses token family rotation with theft detection via missing JTI triggering family-wide revocation (T-03-07)
  - ValidateToken returns only user_id and email from claims, no sensitive data (T-03-09)
metrics:
  duration: 5m
  completed: 2026-04-12T04:40:42Z
  tasks_completed: 2
  tasks_total: 2
  test_count: 32
  files_created: 10
  files_modified: 3
---

# Phase 03 Plan 02: Auth Session Service Layer Summary

Complete auth service-layer business logic with bcrypt login, JWT token rotation with theft detection, and session management via Redis

## What Was Done

### Task 1: Login, RefreshToken, ValidateToken (TDD)

- **Login**: Authenticates user by email+password using bcrypt, normalizes email, generates new token family UUID, issues access+refresh tokens, stores refresh in Redis with 168h TTL
- **RefreshToken**: Parses refresh JWT, verifies JTI exists in Redis (rotation). If JTI missing but JWT valid: theft detected -- revokes entire token family via DeleteTokenFamily. Normal path: deletes old JTI, issues new tokens with same family
- **ValidateToken**: Parses access JWT, returns (user_id, email) only -- no sensitive data exposure

### Task 2: Logout, LogoutAll, Register auto-login (TDD)

- **Logout**: Parses refresh JWT, deletes its JTI from Redis
- **LogoutAll**: Delegates to DeleteAllUserTokens for user-wide session revocation
- **Register**: Updated signature from `(*User, error)` to `(*User, string, string, error)` -- now generates tokens after user creation for auto-login. Updated gRPC handler to populate TokenPair in RegisterResponse

## Test Coverage

| Method | Tests | Key Scenarios |
|--------|-------|---------------|
| Login | 5 | success, non-existent email, wrong password, store refresh, email normalization |
| RefreshToken | 5 | success rotation, theft detection, invalid JWT, expired JWT, delete+store verification |
| ValidateToken | 3 | success, expired, invalid |
| Logout | 2 | success with JTI deletion, invalid token |
| LogoutAll | 2 | success, returns nil |
| Register | 8 | success with tokens, invalid email (4), weak password, duplicates (2), storage error, normalization, store refresh |

Total: 25 new/updated tests, all passing

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed gRPC handler build error from Register signature change**
- **Found during:** Task 2
- **Issue:** `auth/internal/api/auth_service_api/register.go` expected 2 return values from Register but new signature returns 4
- **Fix:** Updated handler to destructure all 4 values and populate TokenPair in RegisterResponse
- **Files modified:** `auth/internal/api/auth_service_api/register.go`
- **Commit:** 43711ff

## Threat Mitigations Applied

| Threat ID | Mitigation | Verified |
|-----------|-----------|----------|
| T-03-06 | Login returns ErrInvalidCredentials for both wrong email and wrong password | grep confirms both paths return same error |
| T-03-07 | Token theft detection: valid JWT + missing JTI triggers DeleteTokenFamily | grep confirms DeleteTokenFamily call |
| T-03-08 | Token rotation: old JTI deleted before new one issued | DeleteRefreshToken called before StoreRefreshToken |
| T-03-09 | ValidateToken returns only Subject (user_id) and Email | code returns claims.Subject, claims.Email only |

## Commits

| Commit | Type | Description |
|--------|------|-------------|
| 875a7c1 | test | Failing tests for Login, RefreshToken, ValidateToken (TDD RED) |
| e763c6a | feat | Implement Login, RefreshToken, ValidateToken (TDD GREEN) |
| 3013e8e | test | Failing tests for Logout and LogoutAll (TDD RED) |
| 43711ff | feat | Implement Logout, LogoutAll, Register auto-login + handler fix (TDD GREEN) |
