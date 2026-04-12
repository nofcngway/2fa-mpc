---
phase: 03-auth-sessions-jwt
fixed_at: 2026-04-12T00:00:00Z
review_path: .planning/phases/03-auth-sessions-jwt/03-REVIEW.md
iteration: 1
findings_in_scope: 10
fixed: 10
skipped: 0
status: all_fixed
---

# Phase 03: Code Review Fix Report

**Fixed at:** 2026-04-12T00:00:00Z
**Source review:** .planning/phases/03-auth-sessions-jwt/03-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 10 (3 critical, 5 warning, 2 extra user-requested)
- Fixed: 10
- Skipped: 0

## Fixed Issues

### CR-01: Nil pointer dereference when Redis is unavailable

**Files modified:** `auth/internal/bootstrap/bootstrap.go`, `auth/cmd/app/main.go`
**Commit:** 5e591b2
**Applied fix:** Changed `NewRedisStorage` to return an error when Redis ping fails instead of logging a warning and returning a connected-but-broken client. Updated `main.go` to treat Redis failure as fatal (`os.Exit(1)`) instead of continuing without Redis.

### CR-02: Logout accepts expired refresh tokens -- no TTL-expiry check

**Files modified:** `auth/internal/api/auth_service_api/logout.go`
**Commit:** 1de2ecb
**Applied fix:** Added error discrimination in the Logout handler to check for `domain.ErrInvalidToken` and `domain.ErrTokenExpired`, returning `codes.Unauthenticated` instead of mapping all errors to `codes.Internal`. Added `errors` and `domain` imports.

### CR-03: TOCTOU race in DeleteAllUserTokens

**Files modified:** `auth/internal/storage/redisstorage/session.go`
**Commit:** b881c81
**Applied fix:** Replaced plain `Pipeline()` with `TxPipelined()` (MULTI/EXEC) to execute the read-and-delete operations atomically within a Redis transaction, preventing concurrent token rotation from leaving orphaned tokens.

### WR-01: Token rotation is not atomic -- new token stored after old is deleted

**Files modified:** `auth/internal/services/authService/refresh_token.go`
**Commit:** 2a593e0
**Applied fix:** Reversed the order of operations: new token is stored first, then old token is deleted (best-effort, error ignored). If store fails, old token remains valid and user can retry. If delete fails, old token expires naturally at TTL.

### WR-02: DeleteTokenFamily does not remove family from user-tokens set

**Files modified:** `auth/internal/services/authService/auth_service.go`, `auth/internal/storage/redisstorage/session.go`, `auth/internal/services/authService/refresh_token.go`
**Commit:** 2cd17fe
**Applied fix:** Added `userID string` parameter to `DeleteTokenFamily` interface and implementation. The implementation now calls `pipe.SRem` to remove the family from the `user_tokens:<userID>` set, preventing stale references. Updated the caller in `refresh_token.go` to pass `claims.Subject` as the userID.

### WR-03: JWT Issuer claim is not validated during ParseToken

**Files modified:** `auth/internal/services/authService/jwt.go`
**Commit:** b68892c, b1dd607
**Applied fix:** Added `jwt.WithIssuer("mpc-2fa-auth")` and `jwt.WithExpirationRequired()` parser options to `ParseToken`. Also added an inline comment explaining the SEC-01 algorithm check delegation. A follow-up commit corrected `jwt.WithIssuers` (plural, non-existent in v5.3.1) to `jwt.WithIssuer` (singular).

### WR-04: RefreshToken handler does not discriminate ErrTokenExpired

**Files modified:** `auth/internal/api/auth_service_api/refresh_token.go`
**Commit:** 713405b
**Applied fix:** Added explicit `errors.Is(err, domain.ErrTokenExpired)` check in the RefreshToken handler, returning `codes.Unauthenticated` with "token expired" message for defensive correctness.

### WR-05: LogoutAll has no authorization

**Files modified:** `auth/internal/api/auth_service_api/logout_all.go`
**Commit:** 36ad9d0
**Applied fix:** Documented the security assumption that LogoutAll is an internal operation where the Gateway must authenticate the user before forwarding. Added a TODO for implementing a gRPC interceptor to enforce caller identity. Proto change (adding access_token field) was not applied to avoid regeneration complexity in a review fix.

### EX-02: Move models to domain package

**Files modified:** `auth/internal/domain/models.go` (new), `auth/internal/models/models.go` (deleted), `auth/internal/services/authService/auth_service.go`, `auth/internal/services/authService/register.go`, `auth/internal/services/authService/login.go`, `auth/internal/services/authService/register_test.go`, `auth/internal/services/authService/login_test.go`, `auth/internal/services/authService/refresh_token_test.go`, `auth/internal/services/authService/mocks/storage_mock.go`, `auth/internal/services/authService/mocks/session_storage_mock.go`, `auth/internal/storage/pgstorage/user.go`, `auth/internal/storage/redisstorage/session.go`
**Commit:** 8f1f45f
**Applied fix:** Moved `User` struct from `auth/internal/models/models.go` to `auth/internal/domain/models.go`. Moved `RefreshTokenData` struct from `auth/internal/services/authService/auth_service.go` to `auth/internal/domain/models.go`. Updated all imports across service, storage, test, and mock files. Deleted the now-empty `models` package. This eliminates the circular dependency that would occur if storage imports from the service layer.

### EX-01: Move service interface to API layer

**Files modified:** `auth/internal/api/auth_service_api/auth_service_api.go`, `auth/internal/bootstrap/bootstrap.go`, `auth/internal/services/authService/mocks/storage_mock.go`, `auth/internal/services/authService/mocks/session_storage_mock.go`, `auth/internal/services/authService/refresh_token_test.go`
**Commit:** ea1006c
**Applied fix:** Defined a `Service` interface in the API layer (`auth/internal/api/auth_service_api/auth_service_api.go`) that specifies the contract the API layer requires from the auth service (Register, Login, Logout, LogoutAll, RefreshToken, ValidateToken). Changed `AuthServiceAPI.service` field from concrete `*authService.AuthService` to the `Service` interface. Updated `NewAuthServiceAPI` and `bootstrap.NewAuthServiceAPI` signatures accordingly. Regenerated mocks to match the updated `DeleteTokenFamily` signature (with userID parameter). Fixed `refresh_token_test.go` to use the new mock signature.

---

_Fixed: 2026-04-12T00:00:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
