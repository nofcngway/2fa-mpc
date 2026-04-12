---
phase: 03-auth-sessions-jwt
fixed_at: 2026-04-12T12:30:00Z
review_path: .planning/phases/03-auth-sessions-jwt/03-REVIEW.md
iteration: 2
findings_in_scope: 5
fixed: 5
skipped: 0
status: all_fixed
---

# Phase 03: Code Review Fix Report

**Fixed at:** 2026-04-12T12:30:00Z
**Source review:** .planning/phases/03-auth-sessions-jwt/03-REVIEW.md
**Iteration:** 2

**Summary:**
- Findings in scope: 5
- Fixed: 5
- Skipped: 0

## Fixed Issues

### WR-01: DeleteAllUserTokens SMembers reads execute outside the MULTI/EXEC boundary

**Files modified:** `auth/internal/storage/redisstorage/session.go`
**Commit:** dff37f8
**Applied fix:** Replaced the TxPipelined implementation with a Lua script (`deleteAllScript`) that atomically reads all families and JTIs, deletes all refresh tokens, family sets, and the user-tokens set in a single server-side operation. This eliminates the TOCTOU race where a concurrent `StoreRefreshToken` could add a new JTI between SMembers reads and Del execution.

### WR-02: user_tokens set grows unbounded with stale family references

**Files modified:** `auth/internal/storage/redisstorage/session.go`
**Commit:** c42990e
**Applied fix:** Added `pipe.Expire(ctx, prefixUserTokens+userID, ttl)` in `StoreRefreshToken` to set a TTL on the `user_tokens:{userID}` set, refreshed on each login. The TTL matches the refresh token TTL, so orphaned family references expire naturally when the user stops logging in, capping growth to families created within one TTL window.

### WR-03: LogoutAll RPC accepts arbitrary user_id without caller authentication

**Files modified:** `auth/internal/api/auth_service_api/logout_all.go`
**Commit:** 70b2898
**Applied fix:** Replaced the generic TODO comment with a structured `SECURITY(WR-03)` comment documenting that caller authentication is deferred to Phase 9 (Gateway interceptors). Phase 9 will add a gRPC interceptor validating service-to-service tokens or mTLS certificates. No API or proto changes -- this is an architectural concern addressed at the Gateway layer.

### IN-01: Bootstrap is a single monolithic file instead of per-concern files

**Files modified:** `auth/internal/bootstrap/pgstorage.go`, `auth/internal/bootstrap/redisstorage.go`, `auth/internal/bootstrap/auth_service.go`, `auth/internal/bootstrap/auth_service_api.go`, `auth/internal/bootstrap/server.go`
**Commit:** 67e964e
**Applied fix:** Split the monolithic `bootstrap.go` (5 factory functions, 74 lines) into five separate files matching the reference pattern from `medialog/students/internal/bootstrap/`. Each file contains one factory function with its own minimal import block. The original `bootstrap.go` was deleted. Build verified with `go build ./...`.

### IN-02: Register normalizes email after validation instead of before

**Files modified:** `auth/internal/services/authService/register.go`
**Commit:** 90a4d0e
**Applied fix:** Moved email normalization (`strings.ToLower(strings.TrimSpace(email))`) to step 1, before validation. The `validateEmail` call now receives the already-normalized value. Removed redundant normalization in the `GetUserByEmail` call and `User` struct construction. Updated step numbering throughout the function for consistency.

---

_Fixed: 2026-04-12T12:30:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 2_
