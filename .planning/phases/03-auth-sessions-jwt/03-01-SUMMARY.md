---
phase: 03-auth-sessions-jwt
plan: 01
subsystem: auth
tags: [jwt, redis, session-storage, rs256, security]
dependency_graph:
  requires: [phase-02-auth-registration]
  provides: [jwt-helper, session-storage-interface, session-storage-redis, rsa-key-loading]
  affects: [auth-service, bootstrap, main]
tech_stack:
  added: [github.com/golang-jwt/jwt/v5@v5.3.1]
  patterns: [RS256-token-signing, three-key-redis-model, algorithm-confusion-prevention]
key_files:
  created:
    - auth/internal/services/authService/jwt.go
    - auth/internal/services/authService/jwt_test.go
    - auth/internal/storage/redisstorage/session.go
    - auth/internal/services/authService/mocks/session_storage_mock.go
  modified:
    - auth/internal/services/authService/auth_service.go
    - auth/internal/services/authService/register_test.go
    - auth/internal/domain/errors.go
    - auth/api/auth_api/auth_service.proto
    - auth/internal/bootstrap/bootstrap.go
    - auth/cmd/app/main.go
    - auth/Makefile
    - auth/go.mod
    - auth/go.sum
decisions:
  - "RS256 algorithm enforced via jwt.WithValidMethods to prevent algorithm confusion attacks (SEC-01)"
  - "Three-key Redis model for session storage: refresh_token:{jti}, token_family:{family}, user_tokens:{user_id}"
  - "RSA keys loaded once at startup in bootstrap and injected into AuthService"
  - "user_tokens:{user_id} set has no TTL (accepted risk for academic project scope)"
metrics:
  duration: 236s
  completed: "2026-04-12T04:33:27Z"
  tasks_completed: 2
  tasks_total: 2
  files_created: 4
  files_modified: 11
---

# Phase 03 Plan 01: JWT Infrastructure and Redis Session Storage Summary

JWT RS256 token generation/parsing with algorithm confusion prevention, Redis SessionStorage with three-key model (refresh_token/token_family/user_tokens), and domain error contracts for auth flows.

## What Was Done

### Task 1: JWT helper, domain errors, proto update, and AuthService struct update
- **Commit:** d46feca
- Created `jwt.go` with `GenerateAccessToken`, `GenerateRefreshToken`, `ParseToken`, and `LoadRSAKeys`
- RS256 signing enforced via `jwt.WithValidMethods([]string{"RS256"})` -- prevents alg:none and HS256 confusion attacks
- Claims include: sub (userID), email, jti (UUID), iat, exp, iss ("mpc-2fa-auth"), token_family (refresh only)
- Added 4 domain errors: `ErrInvalidCredentials`, `ErrInvalidToken`, `ErrTokenExpired`, `ErrTokenRevoked`
- Expanded `SessionStorage` interface with 5 methods and `RefreshTokenData` struct
- Added `LogoutAll` RPC to auth_service.proto
- Updated `AuthService` struct with `privateKey`, `publicKey`, `accessTokenTTL`, `refreshTokenTTL` fields
- Updated `NewAuthService` constructor signature (breaking change, all callers updated)
- Created 7 JWT tests including algorithm confusion prevention test
- Updated register_test.go for new constructor signature -- all existing tests still pass

### Task 2: Redis SessionStorage implementation, bootstrap wiring, and mock generation
- **Commit:** 00cdde9
- Implemented all 5 `SessionStorage` methods on `RedisStorage`:
  - `StoreRefreshToken`: Pipeline SET + SADD + EXPIRE + SADD
  - `GetRefreshToken`: GET + JSON unmarshal, returns nil for missing tokens
  - `DeleteRefreshToken`: GET data, Pipeline DEL + SREM, cleanup empty family
  - `DeleteTokenFamily`: SMEMBERS + Pipeline DEL all JTIs + DEL family set
  - `DeleteAllUserTokens`: SMEMBERS families, Pipeline DEL all JTIs + families + user set
- Three-key Redis model: `refresh_token:{jti}`, `token_family:{family}`, `user_tokens:{user_id}`
- All delete operations are idempotent (no error if keys already deleted)
- Updated `bootstrap.NewAuthService` to load RSA keys from PEM files via config paths
- Updated `main.go` to handle key loading error
- Generated `SessionStorage` mock via minimock for Plan 02 tests

## Deviations from Plan

None -- plan executed exactly as written.

## Verification Results

1. `go test ./internal/services/authService/ -v -count=1` -- 21 tests PASS (7 JWT + 14 register/password)
2. `go build ./...` -- exits 0
3. `go vet ./...` -- exits 0
4. `grep "WithValidMethods" jwt.go` -- SEC-01 enforcement present
5. `grep -c "func (rs *RedisStorage)" session.go` -- returns 5 (all SessionStorage methods)

## Self-Check: PASSED

All 4 created files verified on disk. Both commit hashes (d46feca, 00cdde9) found in git log.
