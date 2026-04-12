---
phase: 08-twofa-verification-management
reviewed: 2026-04-12T00:00:00Z
depth: standard
files_reviewed: 20
files_reviewed_list:
  - twofa/internal/api/twofa_service_api/disable.go
  - twofa/internal/api/twofa_service_api/status.go
  - twofa/internal/api/twofa_service_api/twofa_service_api.go
  - twofa/internal/api/twofa_service_api/verify.go
  - twofa/internal/crypto/totp/totp.go
  - twofa/internal/crypto/totp/totp_test.go
  - twofa/internal/services/twofaService/disable.go
  - twofa/internal/services/twofaService/disable_test.go
  - twofa/internal/services/twofaService/mocks/session_storage_mock.go
  - twofa/internal/services/twofaService/mocks/storage_mock.go
  - twofa/internal/services/twofaService/retrieve_shares.go
  - twofa/internal/services/twofaService/status.go
  - twofa/internal/services/twofaService/status_test.go
  - twofa/internal/services/twofaService/twofa_service.go
  - twofa/internal/services/twofaService/verify.go
  - twofa/internal/services/twofaService/verify_test.go
  - twofa/internal/storage/pgstorage/twofa_record.go
  - twofa/internal/storage/redisstorage/cleanup.go
  - twofa/internal/storage/redisstorage/otp_counter.go
  - twofa/internal/storage/redisstorage/rate_limit.go
findings:
  critical: 1
  warning: 3
  info: 2
  total: 6
status: issues_found
---

# Phase 08: Code Review Report

**Reviewed:** 2026-04-12
**Depth:** standard
**Files Reviewed:** 20
**Status:** issues_found

## Summary

The phase implements TOTP verification, 2FA disable, and status retrieval with sound architectural choices: share zeroization via `defer`, rate-limiting before validation, OTP reuse prevention via Redis-stored counter, and parallel share retrieval with first-2-wins cancellation. Tests are thorough and use real Shamir shares rather than hand-crafted byte fixtures.

Three issues require attention before this can be considered production-ready:

1. A critical data-integrity bug in the Redis rate-limiter: the `Expire` call's error is silently dropped, which can permanently lock a user out of 2FA if the TTL is never set.
2. The Disable flow performs no OTP reuse check — a valid TOTP code captured by an attacker can be replayed within its 30-second (±1) window to disable 2FA.
3. The OTP reuse guard in `Verify` incorrectly allows reuse of any code whose counter resolves to 0.

---

## Critical Issues

### CR-01: Rate-limit TTL silently ignored — user can be permanently locked out

**File:** `twofa/internal/storage/redisstorage/rate_limit.go:18`

**Issue:** `rs.client.Expire(ctx, key, ttl)` is called without capturing or checking its returned `*redis.IntCmd`. If `Expire` fails (e.g., a transient Redis error, key evicted between `Incr` and `Expire`), the key persists with no TTL. Every subsequent `Incr` increments an immortal counter. Once the user exceeds 5 attempts during that session, the rate limit never resets and they are permanently locked out of 2FA verification.

**Fix:**
```go
// IncrementRateLimit atomically increments a rate limit counter and sets TTL on first increment.
func (rs *RedisStorage) IncrementRateLimit(ctx context.Context, key string, ttl time.Duration) (int64, error) {
    count, err := rs.client.Incr(ctx, key).Result()
    if err != nil {
        return 0, fmt.Errorf("incr rate limit: %w", err)
    }
    if count == 1 {
        if err := rs.client.Expire(ctx, key, ttl).Err(); err != nil {
            // Best-effort: delete the key so it does not persist without TTL.
            _ = rs.client.Del(ctx, key).Err()
            return 0, fmt.Errorf("set rate limit ttl: %w", err)
        }
    }
    return count, nil
}
```

The `Del` fallback ensures the counter does not survive without expiry. Returning an error here is consistent with the caller's D-07 resilience path (log and proceed), so the user is not immediately blocked.

---

## Warnings

### WR-01: Disable path skips OTP reuse prevention — captured code can be replayed

**File:** `twofa/internal/services/twofaService/disable.go:52`

**Issue:** `Disable` calls `totp.ValidateOTP(secret, otpCode)` (which uses the plain `validateOTPAt`), not `totp.ValidateOTPWithCounter`. This means the matched counter is never retrieved and never stored in Redis. An attacker who observes or captures a valid 6-digit TOTP code during the ±30-second window can replay it to disable 2FA — the service has no way to detect the reuse.

The `Verify` path correctly uses `ValidateOTPWithCounter` + Redis storage. Disable should follow the same pattern.

**Fix:**
```go
// In disable.go, replace ValidateOTP with ValidateOTPWithCounter:
valid, matchedCounter := totp.ValidateOTPWithCounter(secret, otpCode)
if !valid {
    return fmt.Errorf("disable 2fa: invalid OTP code")
}

// Check reuse using the same Redis key as Verify.
otpUsedKey := fmt.Sprintf("otp_used:%s", userID)
lastCounter, err := s.sessionStorage.GetUsedOTPCounter(ctx, userID)
if err == nil && lastCounter == matchedCounter && matchedCounter != 0 {
    return ErrOTPReused
}

// Store the used counter to prevent replay.
_ = s.sessionStorage.SetUsedOTPCounter(ctx, userID, matchedCounter, otpCounterTTL)
```

### WR-02: OTP reuse check exempts counter value 0 — codes at epoch can be replayed

**File:** `twofa/internal/services/twofaService/verify.go:103`

**Issue:** The reuse check is:
```go
if hasLastCounter && lastCounter == matchedCounter && lastCounter != 0 {
```

The `lastCounter != 0` guard was intended to distinguish "Redis returned 0 because the key doesn't exist" from a genuine counter-0 match. However `GetUsedOTPCounter` already returns `0, nil` for a missing key, and the code separately tracks `hasLastCounter` to reflect whether the call succeeded. The `lastCounter != 0` guard is therefore redundant AND dangerous: any TOTP code whose 30-second window corresponds to counter 0 (unix time 0–29, i.e. the epoch) can be replayed indefinitely.

While unix-epoch timestamps are unreachable in practice today, this is a latent correctness bug: the guard should be removed because `hasLastCounter` already correctly distinguishes the "no prior entry" case.

**Fix:**
```go
// Remove the lastCounter != 0 condition — hasLastCounter already handles the
// "key not present" case (GetUsedOTPCounter returns 0, nil for missing keys,
// but hasLastCounter is only true when the call succeeded).
if hasLastCounter && lastCounter == matchedCounter {
    return false, false, ErrOTPReused
}
```

### WR-03: `EnableTwoFA` silently ignores zero rows-affected

**File:** `twofa/internal/storage/pgstorage/twofa_record.go:39-46`

**Issue:** `EnableTwoFA` issues an UPDATE without checking `CommandTag.RowsAffected()`. If the `twofa_records` row does not exist (e.g., deleted by a concurrent request between `GetTwoFARecord` and `EnableTwoFA`), the UPDATE silently applies to 0 rows, but the method returns `nil`. The caller in `verify.go` then returns `true, true, nil` — signaling successful first-activation — when in fact nothing was persisted.

**Fix:**
```go
func (ps *PGStorage) EnableTwoFA(ctx context.Context, userID string) error {
    query := `UPDATE twofa_records SET is_enabled = TRUE WHERE user_id = $1`
    tag, err := ps.pool.Exec(ctx, query, userID)
    if err != nil {
        return fmt.Errorf("enable twofa: %w", err)
    }
    if tag.RowsAffected() == 0 {
        return fmt.Errorf("enable twofa: record not found for user %s", userID)
    }
    return nil
}
```

---

## Info

### IN-01: `disable.go` error message for invalid OTP is not a sentinel — callers cannot distinguish it

**File:** `twofa/internal/services/twofaService/disable.go:54`

**Issue:** When the OTP is invalid, `Disable` returns `fmt.Errorf("disable 2fa: invalid OTP code")` — a plain string-formatted error, not a exported sentinel. The handler in `api/disable.go` has no `errors.Is` case for this and falls through to the `Internal` gRPC status, logging the failure as an internal error. The client receives a misleading `Internal` error instead of `InvalidArgument` or `Unauthenticated`.

**Fix:** Define and return a sentinel error, then handle it in the API layer:
```go
// In verify.go or disable.go alongside other sentinels:
var ErrInvalidOTP = errors.New("2fa: invalid OTP code")

// In disable.go line 54:
return ErrInvalidOTP

// In api/disable.go, add a case:
case errors.Is(err, twofaService.ErrInvalidOTP):
    return nil, status.Error(codes.InvalidArgument, "invalid OTP code")
```

### IN-02: `validateOTPAt` short-circuits on the first matching window, leaking timing via branch order

**File:** `twofa/internal/crypto/totp/totp.go:56-67`

**Issue:** The three window checks (current, next, previous) return `true` immediately on the first match. An attacker performing remote timing measurements could distinguish whether the code matched the current window vs the previous window by the number of HMAC-SHA1 computations performed. This is a very weak signal over a network, but it contradicts the `subtle.ConstantTimeCompare` intent.

For the context of this academic project the practical risk is low; however, the correct pattern is to compute all three comparisons unconditionally and OR the results:

```go
func validateOTPAt(secret []byte, code string, unixTime int64) bool {
    if len(code) != 6 {
        return false
    }
    for _, c := range code {
        if c < '0' || c > '9' {
            return false
        }
    }
    counter := uint64(unixTime) / 30
    codeBytes := []byte(code)

    cur  := subtle.ConstantTimeCompare([]byte(hotp(secret, counter)),   codeBytes)
    next := subtle.ConstantTimeCompare([]byte(hotp(secret, counter+1)), codeBytes)
    var prev int
    if counter > 0 {
        prev = subtle.ConstantTimeCompare([]byte(hotp(secret, counter-1)), codeBytes)
    }
    return (cur | next | prev) == 1
}
```

The same refactor applies to `validateOTPWithCounterAt`, though that function must retain the matched-counter return value, requiring a slightly different structure.

---

_Reviewed: 2026-04-12_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
