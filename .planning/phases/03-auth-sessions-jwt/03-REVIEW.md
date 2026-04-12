---
phase: 03-auth-sessions-jwt
reviewed: 2026-04-12T00:00:00Z
depth: standard
files_reviewed: 18
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
  - auth/internal/domain/errors.go
  - auth/internal/bootstrap/bootstrap.go
  - auth/internal/api/auth_service_api/login.go
  - auth/internal/api/auth_service_api/refresh_token.go
  - auth/internal/api/auth_service_api/validate_token.go
  - auth/internal/api/auth_service_api/logout.go
  - auth/internal/api/auth_service_api/logout_all.go
  - auth/internal/api/auth_service_api/register.go
  - auth/cmd/app/main.go
findings:
  critical: 3
  warning: 5
  info: 4
  total: 12
status: issues_found
---

# Phase 03: Code Review Report

**Reviewed:** 2026-04-12T00:00:00Z
**Depth:** standard
**Files Reviewed:** 18
**Status:** issues_found

## Summary

Phase 03 implements JWT RS256 token issuance, three-key Redis session model, token-family theft detection, and all auth CRUD handlers. The architecture is clean and the algorithm confusion prevention (SEC-01) is correctly implemented via `jwt.WithValidMethods([]string{"RS256"})`. Password hashes are not returned in responses, satisfying SEC-03. Token theft detection logic is conceptually correct.

Three critical issues are present: (1) the service continues to start when Redis is unavailable, making all session operations silently fail against a nil pointer; (2) `Logout` accepts an expired token as valid — a user's expired token can still be "logged out," but more importantly an attacker who obtains an expired refresh token can produce an invalid-logout-attempt-as-signal without hitting a panic; (3) `DeleteAllUserTokens` reads individual family JTI sets outside the pipeline, creating a TOCTOU window during logout-all. Additionally, five warnings cover missing error discrimination in `logout.go`, a non-atomic delete+store in token rotation, missing user-token-set cleanup in `DeleteTokenFamily`, Redis-unavailable silent degradation, and a missing `Issuer` validation during token parsing.

---

## Critical Issues

### CR-01: Nil pointer dereference when Redis is unavailable

**File:** `auth/cmd/app/main.go:36-41` and `auth/internal/bootstrap/bootstrap.go:34-39`

**Issue:** `NewRedisStorage` logs a warning and returns `(rs, nil)` even when Redis is unreachable. `main.go` then passes a non-nil `*redisstorage.RedisStorage` (whose underlying client is disconnected) as the `SessionStorage` argument to `NewAuthService`. Every call to `StoreRefreshToken`, `GetRefreshToken`, etc. will receive a Redis error at runtime rather than a startup failure, meaning Login/Register silently return `codes.Internal` for all users. The `redisStorage != nil` guard in `main.go` (line 37) never triggers because the value is never nil.

**Fix:**
```go
// bootstrap.go — fail hard, do not silently swallow Redis failure
func NewRedisStorage(ctx context.Context, cfg *config.Config) (*redisstorage.RedisStorage, error) {
    rs := redisstorage.New(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
    if err := rs.Ping(ctx); err != nil {
        return nil, fmt.Errorf("redis ping failed: %w", err)
    }
    slog.Info("Redis connected")
    return rs, nil
}

// main.go — treat Redis failure as fatal
redisStorage, err := bootstrap.NewRedisStorage(ctx, cfg)
if err != nil {
    slog.Error("failed to connect to Redis", "error", err)
    os.Exit(1)
}
defer redisStorage.Close()
```

---

### CR-02: Logout accepts expired refresh tokens — no TTL-expiry check

**File:** `auth/internal/services/authService/logout.go:11-16`

**Issue:** `Logout` calls `ParseToken`, which returns `domain.ErrTokenExpired` when the JWT is expired. The handler in `logout.go` (service layer) maps any parse error to `domain.ErrInvalidToken` and returns it — but the API handler in `auth/internal/api/auth_service_api/logout.go` maps *all* service errors (including `ErrInvalidToken`) to `codes.Internal`, not `codes.Unauthenticated`. This means a caller presenting an expired (but otherwise structurally valid) refresh token receives a misleading `Internal` error instead of `Unauthenticated`, and the non-existent Redis key is queried unnecessarily. More importantly, the `logout.go` API handler does not discriminate `ErrInvalidToken` at all — it catches every error as `Internal`, which is a wrong gRPC mapping and hides token-validity failures from clients.

**Fix:**
```go
// auth/internal/api/auth_service_api/logout.go
func (api *AuthServiceAPI) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
    if req.RefreshToken == "" {
        return nil, status.Error(codes.InvalidArgument, "refresh token is required")
    }

    if err := api.service.Logout(ctx, req.RefreshToken); err != nil {
        if errors.Is(err, domain.ErrInvalidToken) || errors.Is(err, domain.ErrTokenExpired) {
            return nil, status.Error(codes.Unauthenticated, "invalid token")
        }
        return nil, status.Error(codes.Internal, "internal error")
    }

    return &pb.LogoutResponse{}, nil
}
```

The same fix applies to `LogoutAll`: presenting an empty or invalid `user_id` should return `InvalidArgument`, but sending a `user_id` for a non-existent user currently succeeds silently (acceptable), so no change needed there.

---

### CR-03: TOCTOU race in DeleteAllUserTokens — reads JTI sets outside the pipeline

**File:** `auth/internal/storage/redisstorage/session.go:152-186`

**Issue:** `DeleteAllUserTokens` reads each family's JTI set with a blocking `SMembers` call *inside a loop*, then accumulates `Del` commands in a pipeline. Between the `SMembers` read and the pipeline `Exec`, another goroutine (e.g., concurrent `RefreshToken`) can add a new JTI to the same family set. The new JTI escapes deletion, leaving an orphaned refresh token in Redis that remains valid until its own TTL expires. Under multi-device concurrent refresh this can produce sessions that survive `LogoutAll`.

**Fix:** Use `MULTI`/`EXEC` (Redis transactions) or a Lua script to read-and-delete atomically. With the `go-redis` client, `TxPipelined` provides `MULTI`/`EXEC`:

```go
func (rs *RedisStorage) DeleteAllUserTokens(ctx context.Context, userID string) error {
    families, err := rs.client.SMembers(ctx, prefixUserTokens+userID).Result()
    if err != nil && err != redis.Nil {
        return fmt.Errorf("get user families: %w", err)
    }

    _, err = rs.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
        for _, family := range families {
            jtis, err := rs.client.SMembers(ctx, prefixTokenFamily+family).Result()
            if err != nil && err != redis.Nil {
                return fmt.Errorf("get family %s members: %w", family, err)
            }
            for _, jti := range jtis {
                pipe.Del(ctx, prefixRefreshToken+jti)
            }
            pipe.Del(ctx, prefixTokenFamily+family)
        }
        pipe.Del(ctx, prefixUserTokens+userID)
        return nil
    })
    return err
}
```

Note: even `TxPipelined` only guarantees serial execution on a single Redis node; for a clustered Redis deployment a Lua script would be required.

---

## Warnings

### WR-01: Token rotation is not atomic — new token stored after old is deleted

**File:** `auth/internal/services/authService/refresh_token.go:33-51`

**Issue:** Steps 4 (delete old JTI) through 7 (store new JTI) are executed as three separate Redis round-trips. If the service crashes or the Redis write fails between steps 4 and 7, the old token is deleted but no new token is issued. The user is permanently logged out with no ability to re-authenticate without their password. This is a correctness issue under failure conditions.

**Fix:** Reverse the order — store the new token first, then delete the old one. If the store fails, the old token remains valid and the user can retry. If the delete fails after a successful store, the old token expires naturally at its TTL boundary (7 days). This is the safer failure mode:

```go
// 5. Generate new tokens
newAccess, _, err := s.GenerateAccessToken(tokenData.UserID, claims.Email)
if err != nil { return "", "", err }

newRefresh, newJTI, err := s.GenerateRefreshToken(tokenData.UserID, claims.Email, tokenData.TokenFamily)
if err != nil { return "", "", err }

// 6. Store new token BEFORE deleting old
if err := s.sessionStorage.StoreRefreshToken(ctx, newJTI, tokenData.UserID, tokenData.TokenFamily, s.refreshTokenTTL); err != nil {
    return "", "", err
}

// 7. Delete old token (best-effort — it will expire naturally on TTL if this fails)
_ = s.sessionStorage.DeleteRefreshToken(ctx, claims.ID)

return newAccess, newRefresh, nil
```

---

### WR-02: DeleteTokenFamily does not remove the family from the user-tokens set

**File:** `auth/internal/storage/redisstorage/session.go:117-148`

**Issue:** `DeleteTokenFamily` deletes individual `refresh_token:<jti>` keys and the `token_family:<family>` set but does NOT call `SRem(prefixUserTokens+userID, family)`. This means after theft detection triggers `DeleteTokenFamily`, the `user_tokens:<userID>` set retains a stale family reference. A subsequent `DeleteAllUserTokens` call iterates over these ghost families, calling `SMembers` on empty/non-existent keys — functionally harmless today but a memory leak in Redis for long-lived accounts that experience many theft-detection events. The user-token set also needs the user ID, which `DeleteTokenFamily` does not have.

**Fix:** Propagate `userID` into `DeleteTokenFamily` so the stale reference can be cleaned up, OR read the first JTI's token data before deleting to obtain the `userID`:

```go
func (rs *RedisStorage) DeleteTokenFamily(ctx context.Context, family string) error {
    jtis, err := rs.client.SMembers(ctx, prefixTokenFamily+family).Result()
    // ... existing logic ...

    // After deleting all JTIs, look up userID from any token data we fetched
    // and remove the family from the user set.
    // Requires either passing userID as parameter, or reading one token before deletion.
    if userID != "" {
        pipe.SRem(ctx, prefixUserTokens+userID, family)
    }
    // ...
}
```

The interface signature `DeleteTokenFamily(ctx context.Context, family string)` should add `userID string` as a parameter. Update callers in `refresh_token.go` to pass `tokenData.TokenFamily` and the `userID` from `tokenData.UserID` (which is available at the call site when theft is detected).

---

### WR-03: JWT Issuer claim is not validated during ParseToken

**File:** `auth/internal/services/authService/jwt.go:77-96`

**Issue:** Tokens are issued with `Issuer: "mpc-2fa-auth"` but `ParseToken` does not enforce the issuer claim using `jwt.WithIssuers(...)`. A token generated by a different system (e.g., another service sharing the same public key, or a test harness) would pass validation and be accepted as a valid auth token, potentially allowing cross-service token reuse.

**Fix:**
```go
token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
    return s.publicKey, nil
},
    jwt.WithValidMethods([]string{"RS256"}),
    jwt.WithIssuers("mpc-2fa-auth"),
    jwt.WithExpirationRequired(),
)
```

---

### WR-04: RefreshToken handler does not discriminate ErrTokenExpired

**File:** `auth/internal/api/auth_service_api/refresh_token.go:23-31`

**Issue:** `RefreshToken` service layer can return `domain.ErrTokenExpired` (when the refresh JWT itself has expired) via the `ParseToken` call at line 13 of `refresh_token.go`. The service then maps this to `domain.ErrInvalidToken` (line 17). The API handler correctly maps `ErrInvalidToken` to `codes.Unauthenticated`, so the end result is correct. However, if the service layer is changed to bubble `ErrTokenExpired` directly, the API handler would fall through to `codes.Internal`. The handler should also handle `ErrTokenExpired` explicitly for defensive correctness:

```go
if errors.Is(err, domain.ErrTokenExpired) {
    return nil, status.Error(codes.Unauthenticated, "token expired")
}
```

---

### WR-05: LogoutAll has no authorization — any caller can revoke any user's sessions

**File:** `auth/internal/api/auth_service_api/logout_all.go:13-23`

**Issue:** `LogoutAll` accepts a plain `user_id` string and immediately revokes all sessions for that user with no verification that the caller owns that `user_id`. Any gRPC client that can reach the auth service can call `LogoutAll("any-uuid")` and log out an arbitrary user. This is an authorization bypass.

This may be intentional if `LogoutAll` is an internal admin operation only callable from within the cluster (Gateway enforcing auth before forwarding). If so, it should be documented explicitly and protected by a gRPC interceptor that validates the caller is the Gateway or an admin token. If it is user-facing (called after the user authenticates), the handler must receive a validated access token and extract the `user_id` from its claims rather than trusting the request payload.

**Fix (user-facing approach):**
```go
// Accept access token, extract user_id from validated claims
func (api *AuthServiceAPI) LogoutAll(ctx context.Context, req *pb.LogoutAllRequest) (*pb.LogoutAllResponse, error) {
    if req.AccessToken == "" {
        return nil, status.Error(codes.InvalidArgument, "access token is required")
    }
    userID, _, err := api.service.ValidateToken(ctx, req.AccessToken)
    if err != nil {
        return nil, status.Error(codes.Unauthenticated, "invalid token")
    }
    if err := api.service.LogoutAll(ctx, userID); err != nil {
        return nil, status.Error(codes.Internal, "internal error")
    }
    return &pb.LogoutAllResponse{}, nil
}
```

---

## Info

### IN-01: ParseToken keyfunc ignores the token parameter — minor but non-idiomatic

**File:** `auth/internal/services/authService/jwt.go:80-82`

**Issue:** The keyfunc `func(token *jwt.Token) (interface{}, error)` ignores its `token` argument entirely. Since algorithm validation is delegated to `jwt.WithValidMethods`, this is safe, but the conventional Go idiom still performs an in-keyfunc algorithm check as a defense-in-depth guard. The current code is correct (the `WithValidMethods` option provides the protection), but worth a comment explaining the deliberate choice.

**Fix:** Add an inline comment:
```go
func(token *jwt.Token) (interface{}, error) {
    // Algorithm check is delegated to jwt.WithValidMethods above (SEC-01).
    return s.publicKey, nil
},
```

---

### IN-02: Commented-out warning log in `DeleteTokenFamily` line 129

**File:** `auth/internal/storage/redisstorage/session.go:128-130`

**Issue:** The `rs.client.Del(...)` call on line 129 (inside the `len(jtis) == 0` branch) discards its return value and error silently. This is a fire-and-forget Redis call in a non-critical path, but it is inconsistent with the rest of the file where errors are always checked. If the empty-family `Del` fails, no diagnostic is surfaced.

**Fix:**
```go
if _, err := rs.client.Del(ctx, prefixTokenFamily+family).Result(); err != nil {
    slog.Warn("failed to delete empty family key", "family", family, "error", err)
}
```

---

### IN-03: `validateEmail` in register.go does not normalize before validation

**File:** `auth/internal/services/authService/register.go:88-107`

**Issue:** `validateEmail` trims the email on line 89, but the email passed into it at line 25 is the raw `email` argument that has not yet been normalized. After `validateEmail` returns (line 26), the normalized email is computed again at lines 35 and 53 with `strings.ToLower(strings.TrimSpace(email))`. This means validation is run on the un-normalized value (though `TrimSpace` is applied inside `validateEmail`), while the lowercase normalization only happens later. An email with uppercase letters passes validation fine, but if `validateEmail` ever adds a case-sensitive check it would behave unexpectedly. Minor: normalize first, then validate.

**Fix:**
```go
// 1. Normalize email first
email = strings.ToLower(strings.TrimSpace(email))

// 2. Validate the normalized form
if err := validateEmail(email); err != nil {
    return nil, "", "", err
}
// 3. No need to normalize again below — use email directly
```

---

### IN-04: `StoreRefreshToken` pipeline sets family-set TTL but not user-tokens-set TTL

**File:** `auth/internal/storage/redisstorage/session.go:39-44`

**Issue:** `prefixTokenFamily+tokenFamily` gets an `Expire` call with the refresh token TTL. `prefixUserTokens+userID` gets no TTL — it grows indefinitely as the user creates sessions. For users with many login events over months, this set accumulates entries. `DeleteAllUserTokens` does clean it up, but a user who never explicitly logs out all sessions will have a permanently growing set. Per D-05 in the comments this was intentional ("no TTL per D-05"), but it is worth noting that stale families (from tokens already expired) remain referenced in the user set until `DeleteAllUserTokens` is called. If this is a known trade-off, a comment should say so; if not, periodic cleanup or a shorter TTL on the user set should be considered.

**Fix:** Document the design choice explicitly in code, or implement periodic cleanup:
```go
// Add family to user tokens set.
// NOTE (D-05): No TTL on user_tokens set by design — cleaned up on LogoutAll.
// Stale entries from expired tokens accumulate until then; acceptable for this threat model.
pipe.SAdd(ctx, prefixUserTokens+userID, tokenFamily)
```

---

_Reviewed: 2026-04-12T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
