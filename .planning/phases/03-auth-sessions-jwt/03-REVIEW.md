---
phase: 03-auth-sessions-jwt
reviewed: 2026-04-12T12:00:00Z
depth: standard
files_reviewed: 20
files_reviewed_list:
  - auth/internal/services/authService/jwt.go
  - auth/internal/services/authService/auth_service.go
  - auth/internal/services/authService/login.go
  - auth/internal/services/authService/refresh_token.go
  - auth/internal/services/authService/validate_token.go
  - auth/internal/services/authService/logout.go
  - auth/internal/services/authService/logout_all.go
  - auth/internal/services/authService/register.go
  - auth/internal/storage/redisstorage/session.go
  - auth/internal/storage/redisstorage/redisstorage.go
  - auth/internal/domain/errors.go
  - auth/internal/domain/models.go
  - auth/internal/bootstrap/bootstrap.go
  - auth/internal/api/auth_service_api/auth_service_api.go
  - auth/internal/api/auth_service_api/login.go
  - auth/internal/api/auth_service_api/refresh_token.go
  - auth/internal/api/auth_service_api/validate_token.go
  - auth/internal/api/auth_service_api/logout.go
  - auth/internal/api/auth_service_api/logout_all.go
  - auth/internal/api/auth_service_api/register.go
  - auth/cmd/app/main.go
findings:
  critical: 0
  warning: 3
  info: 2
  total: 5
status: issues_found
---

# Phase 03: Code Review Report (Re-review)

**Reviewed:** 2026-04-12T12:00:00Z
**Depth:** standard
**Files Reviewed:** 20
**Status:** issues_found

## Summary

Re-review after fixes from the initial code review. All three critical issues from the first review have been resolved: Redis now fails hard at startup on connection failure (CR-01), Logout API handler properly discriminates token errors (CR-02), and `DeleteAllUserTokens` uses `TxPipelined` (CR-03 partial). Most warnings were also addressed: token rotation now stores-before-deleting (WR-01), `DeleteTokenFamily` now accepts `userID` and cleans user-tokens set (WR-02), JWT issuer validation is enforced (WR-03), and `RefreshToken` API handler handles `ErrTokenExpired` (WR-04).

Three remaining warnings and two info items are identified below. No new bugs were introduced by the fixes.

## Warnings

### WR-01: DeleteAllUserTokens SMembers reads execute outside the MULTI/EXEC boundary

**File:** `auth/internal/storage/redisstorage/session.go:157-190`
**Issue:** The comment on line 156 states "Uses MULTI/EXEC (TxPipelined) to avoid TOCTOU races with concurrent token rotation." However, the `SMembers` call on line 159 reads families using `rs.client.SMembers` (the regular client, not the pipeline), executing before MULTI. Inside the `TxPipelined` callback, line 169 also calls `rs.client.SMembers` to read family JTIs -- this again uses the regular client, not the pipeliner. Redis `MULTI/EXEC` only guarantees atomicity for commands queued via the pipeliner. The reads happen before those commands are submitted, so a concurrent `StoreRefreshToken` can add a new JTI between the `SMembers` read and the `Del` execution, leaving orphaned refresh tokens that survive `LogoutAll`.

**Fix:** Use a Lua script for true atomicity -- read and delete in a single server-side operation:

```go
var deleteAllScript = redis.NewScript(`
    local families = redis.call('SMEMBERS', KEYS[1])
    for _, family in ipairs(families) do
        local jtis = redis.call('SMEMBERS', 'token_family:' .. family)
        for _, jti in ipairs(jtis) do
            redis.call('DEL', 'refresh_token:' .. jti)
        end
        redis.call('DEL', 'token_family:' .. family)
    end
    redis.call('DEL', KEYS[1])
    return 1
`)

func (rs *RedisStorage) DeleteAllUserTokens(ctx context.Context, userID string) error {
    err := deleteAllScript.Run(ctx, rs.client, []string{prefixUserTokens + userID}).Err()
    if err != nil && err != redis.Nil {
        return fmt.Errorf("delete all user tokens: %w", err)
    }
    return nil
}
```

### WR-02: user_tokens set grows unbounded with stale family references

**File:** `auth/internal/storage/redisstorage/session.go:43-44`
**Issue:** The `user_tokens:{userID}` set has no TTL (line 43 comment: "no TTL per D-05"). When a user logs in, a new token family is added. The family is removed only on explicit logout or when `DeleteRefreshToken` detects an empty family set (lines 97-111). If a user simply lets their refresh token expire without logging out, the `token_family:{family}` key expires via its TTL, but the family string remains in `user_tokens:{userID}` permanently. Over many login cycles, this set accumulates stale entries. `DeleteAllUserTokens` iterates all of them, issuing `SMembers` on non-existent keys (returning empty sets), wasting Redis round-trips. The set itself never expires and grows indefinitely for active users who never call LogoutAll.

**Fix:** Add `Expire` on the `user_tokens` set, refreshed on each login to the longest possible TTL (refresh token TTL). This caps growth to families created within one TTL window:

```go
pipe.SAdd(ctx, prefixUserTokens+userID, tokenFamily)
pipe.Expire(ctx, prefixUserTokens+userID, ttl) // refresh TTL on each login
```

Alternatively, clean stale entries lazily: when `DeleteAllUserTokens` encounters a family whose `SMembers` returns empty, remove it from the user set before proceeding.

### WR-03: LogoutAll RPC accepts arbitrary user_id without caller authentication

**File:** `auth/internal/api/auth_service_api/logout_all.go:14-16`
**Issue:** The TODO on line 16 acknowledges this: "Add gRPC interceptor to enforce caller identity (e.g., mTLS or service token)." `LogoutAll` accepts a raw `user_id` and immediately revokes all sessions with no verification that the caller is authorized to act on behalf of that user. If the auth service gRPC port is exposed beyond the trusted network (misconfigured firewall, compromised adjacent service), any client can mass-revoke sessions for arbitrary users. This was flagged in the initial review (WR-05) and remains open with only a TODO comment.

**Fix:** Implement one of:
1. gRPC interceptor validating a service-to-service token or mTLS certificate for internal-only RPCs.
2. Require the caller to pass an access token in gRPC metadata; the handler validates it and extracts `user_id` from claims rather than trusting the request payload.
3. At minimum, add the auth service to a network policy that restricts which pods/services can connect to its gRPC port.

## Info

### IN-01: Bootstrap is a single monolithic file instead of per-concern files

**File:** `auth/internal/bootstrap/bootstrap.go`
**Issue:** The project reference pattern (medialog/students at `/Users/vbncursed/programming/medialog/students/internal/bootstrap/`) splits bootstrap into separate files: `pgstorage.go`, `student_service.go`, `students_api.go`, `server.go`. The auth bootstrap consolidates all five factory functions (`NewPGStorage`, `NewRedisStorage`, `NewAuthService`, `NewAuthServiceAPI`, `NewGRPCServer`) into a single 74-line file. While functional, this deviates from the established pattern. As more dependencies are added (Kafka producer, Prometheus metrics registry), the file will grow and become harder to navigate.

**Fix:** Split into separate files matching the reference pattern:
- `auth/internal/bootstrap/pgstorage.go` -- `NewPGStorage`
- `auth/internal/bootstrap/redisstorage.go` -- `NewRedisStorage`
- `auth/internal/bootstrap/auth_service.go` -- `NewAuthService`
- `auth/internal/bootstrap/auth_service_api.go` -- `NewAuthServiceAPI`
- `auth/internal/bootstrap/server.go` -- `NewGRPCServer`

### IN-02: Register normalizes email after validation instead of before

**File:** `auth/internal/services/authService/register.go:24-35`
**Issue:** `validateEmail` is called on line 25 with the raw `email` argument. Inside `validateEmail` (line 88-89), `TrimSpace` is applied but `ToLower` is not. The actual lowercase normalization happens later at line 35 (`strings.ToLower(strings.TrimSpace(email))`) and again at line 53. This means validation runs against the un-lowered value. Currently harmless since `validateEmail` has no case-sensitive checks, but if a case-sensitive rule is ever added it would validate the wrong form. Normalize once at the top, then use the normalized value throughout.

**Fix:**
```go
// 1. Normalize email first
email = strings.ToLower(strings.TrimSpace(email))

// 2. Validate the normalized form
if err := validateEmail(email); err != nil {
    return nil, "", "", err
}

// 3. Use email directly below -- no need to normalize again
```

---

_Reviewed: 2026-04-12T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
