# Phase 8: TwoFA Verification & Management - Research

**Researched:** 2026-04-12
**Domain:** Go gRPC service layer — OTP verification, rate limiting, session management, parallel MPC coordination
**Confidence:** HIGH

## Summary

Phase 8 implements the verification and management side of the TwoFA service. All cryptographic primitives (Shamir Combine, TOTP ValidateOTP), MPC node gRPC communication patterns (errgroup parallelism, compensating deletes), zeroization utilities, and storage infrastructure already exist from Phases 4-7. The primary work is: (1) adding new service methods (Verify, Disable, GetStatus) following the established one-file-per-method pattern, (2) extending Storage and SessionStorage interfaces with new methods, (3) implementing Redis operations for rate limiting and OTP reuse prevention, (4) implementing PostgreSQL operations for EnableTwoFA, DeleteTwoFARecord, DeleteBackupCodes, and (5) wiring the handler stubs to delegate to service layer with proper gRPC error mapping.

A critical finding: the current `ValidateOTP(secret []byte, code string) bool` function does NOT return the matched time counter. OTP reuse prevention (D-09/D-10) requires knowing WHICH counter matched. A new function `ValidateOTPWithCounter` must be added to the TOTP package that returns `(bool, int64)` — the validity flag and the matched counter value. This is the only crypto-layer change needed.

**Primary recommendation:** Follow the established Phase 7 patterns exactly — errgroup for parallel MPC calls, defer zeroize for secrets/shares, minimock for testing, one file per method. The "first 2 wins" retrieval pattern is the only novel concurrency pattern; all other work is straightforward interface extension and CRUD.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Query all 3 MPC nodes in parallel via errgroup. Use first 2 successful responses for Shamir combine. Cancel remaining on 2 successes. Tolerates 1 node being down.
- **D-02:** If only 1 or 0 nodes respond successfully, return `codes.Internal` error.
- **D-03:** When 2 shares succeed but 3rd fails, proceed silently. Log node failure via slog.
- **D-04:** `defer zeroize()` for each retrieved share's Data field AND the combined secret. Use shared zeroize utility from `twofa/internal/crypto/`.
- **D-05:** Redis `INCR` + `EXPIRE` pattern. Key: `rate_limit:verify:{user_id}`. EXPIRE 300s. Max 5 attempts.
- **D-06:** Count ALL verification attempts regardless of outcome.
- **D-07:** When Redis is unavailable, log warning and proceed without rate check.
- **D-08:** SessionStorage interface gets methods: `IncrementRateLimit(ctx, key string, ttl time.Duration) (int64, error)` and `GetRateLimit(ctx, key string) (int64, error)`.
- **D-09:** Store last validated TOTP time counter in Redis. Key: `otp_used:{user_id}`, value: time counter (int64), TTL: 90s.
- **D-10:** Reject if new counter == stored counter (exact match).
- **D-11:** SessionStorage gets methods: `SetUsedOTPCounter(ctx, userID string, counter int64, ttl time.Duration) error` and `GetUsedOTPCounter(ctx, userID string) (int64, error)`.
- **D-12:** Disable2FA order: Verify OTP, delete shares (parallel errgroup), delete backup codes, delete twofa_record.
- **D-13:** If share deletion fails on any MPC node, return `codes.Internal`. twofa_record stays enabled.
- **D-14:** On successful disable, clean up Redis keys: DEL `rate_limit:verify:{user_id}` and `otp_used:{user_id}`.
- **D-15:** SessionStorage gets method: `DeleteKeys(ctx context.Context, keys ...string) error`.
- **D-16:** Storage interface gets: `EnableTwoFA`, `DeleteTwoFARecord`, `DeleteBackupCodes`.
- **D-17:** Get2FAStatus uses existing `GetTwoFARecord`.

### Claude's Discretion
- Exact errgroup cancellation pattern for "first 2 wins" retrieval
- Whether to wrap rate limit check + OTP validation in a single service method or separate
- Prometheus metric labels for verify/disable/status operations
- Kafka audit event structure for `2fa.verified`, `2fa.disabled`, `2fa.status_checked`
- Internal helper decomposition within Verify and Disable methods
- Error message wording (must not leak internal state per SEC-02)

### Deferred Ideas (OUT OF SCOPE)
- Kafka audit events -- Phase 9
- Prometheus metrics -- Phase 9
- Backup code verification as alternative to OTP -- not in current phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| 2FA-03 | User can verify OTP: 2 shares retrieved, Shamir combine, TOTP validation (+-1 window), secret zeroized | "First 2 wins" retrieval pattern + new ValidateOTPWithCounter + existing Combine + existing Zeroize |
| 2FA-04 | First successful verification enables 2FA (is_enabled=true) | New EnableTwoFA storage method, check is_newly_enabled in Verify |
| 2FA-05 | OTP verification rate limited: max 5 attempts per 5 min per user_id | Redis INCR+EXPIRE pattern, IncrementRateLimit/GetRateLimit on SessionStorage |
| 2FA-06 | User can disable 2FA: verify OTP first, then delete shares + metadata | Verify then parallel DeleteShare + sequential PG cleanup |
| 2FA-07 | User can check 2FA status (is_enabled, created_at) | Existing GetTwoFARecord, trivial handler |
| 2FA-09 | OTP single-use enforcement: reject reuse within same window | New ValidateOTPWithCounter returning counter + Redis SetUsedOTPCounter/GetUsedOTPCounter |
| SEC-05 | Share data and encryption keys never logged or included in events | Zeroize patterns, no slog of share/secret data, sanitized error messages |
</phase_requirements>

## Standard Stack

### Core (already in project)
| Library | Version | Purpose | Status |
|---------|---------|---------|--------|
| golang.org/x/sync/errgroup | latest | Parallel MPC node queries with context cancellation | Already used in setup.go [VERIFIED: codebase] |
| github.com/redis/go-redis/v9 | 9.x | Rate limiting, OTP reuse counter storage | Already initialized in redisstorage.go [VERIFIED: codebase] |
| github.com/jackc/pgx/v5 | 5.x | EnableTwoFA, DeleteTwoFARecord, DeleteBackupCodes SQL operations | Already used throughout PGStorage [VERIFIED: codebase] |
| github.com/gojuno/minimock/v3 | 3.x | Mock generation for SessionStorage (new methods) | Already used in setup_test.go [VERIFIED: codebase] |
| gotest.tools/v3/assert | 3.x | Test assertions | Already used in setup_test.go [VERIFIED: codebase] |

### Supporting
| Library | Version | Purpose | Status |
|---------|---------|---------|--------|
| twofa/internal/crypto | n/a | Zeroize utility | Already exists [VERIFIED: codebase] |
| twofa/internal/crypto/shamir | n/a | Combine(shares) for secret reconstruction | Already implemented [VERIFIED: codebase] |
| twofa/internal/crypto/totp | n/a | ValidateOTP (needs extension for counter return) | Exists but needs ValidateOTPWithCounter [VERIFIED: codebase] |

**No new external dependencies required.** All libraries are already in go.mod.

## Architecture Patterns

### File Structure (new/modified files)
```
twofa/
├── internal/
│   ├── crypto/totp/
│   │   └── totp.go                          # ADD: ValidateOTPWithCounter function
│   ├── services/twofaService/
│   │   ├── twofa_service.go                 # MODIFY: extend Storage + SessionStorage interfaces
│   │   ├── verify.go                        # NEW: Verify method
│   │   ├── disable.go                       # NEW: Disable method
│   │   ├── status.go                        # NEW: GetStatus method
│   │   ├── retrieve_shares.go               # NEW: "first 2 wins" retrieval helper
│   │   ├── verify_test.go                   # NEW: tests
│   │   ├── disable_test.go                  # NEW: tests
│   │   ├── status_test.go                   # NEW: tests
│   │   └── mocks/
│   │       └── session_storage_mock.go      # REGENERATE: minimock with new methods
│   ├── api/twofa_service_api/
│   │   ├── twofa_service_api.go             # MODIFY: extend Service interface
│   │   ├── verify.go                        # MODIFY: implement handler
│   │   ├── disable.go                       # MODIFY: implement handler
│   │   └── status.go                        # MODIFY: implement handler
│   └── storage/
│       ├── pgstorage/
│       │   └── twofa_record.go              # MODIFY: add EnableTwoFA, DeleteTwoFARecord, DeleteBackupCodes
│       └── redisstorage/
│           ├── rate_limit.go                # NEW: IncrementRateLimit, GetRateLimit
│           ├── otp_counter.go               # NEW: SetUsedOTPCounter, GetUsedOTPCounter
│           └── cleanup.go                   # NEW: DeleteKeys
```

### Pattern 1: "First 2 Wins" Parallel Share Retrieval
**What:** Query all 3 MPC nodes concurrently, succeed as soon as any 2 respond, cancel the 3rd.
**When to use:** Verify2FA and Disable2FA flows — need minimum 2 shares for Shamir 2-of-3.
**Implementation approach:**

```go
// Source: custom pattern based on errgroup + atomic counter [ASSUMED]
func (s *TwoFAService) retrieveShares(ctx context.Context, userID string) ([]shamir.Share, error) {
    type result struct {
        share shamir.Share
        node  int
    }

    ctx, cancel := context.WithCancel(ctx)
    defer cancel()

    results := make(chan result, 3)
    errs := make(chan error, 3)

    for i, client := range s.mpcClients {
        go func(idx int, c MPCClient) {
            callCtx, callCancel := context.WithTimeout(ctx, s.mpcTimeout)
            defer callCancel()

            resp, err := c.RetrieveShare(callCtx, &mpc_api.RetrieveShareRequest{
                UserId:     userID,
                ShareIndex: int32(idx + 1),
            })
            if err != nil {
                errs <- fmt.Errorf("node %d: %w", idx, err)
                return
            }
            results <- result{
                share: shamir.Share{Index: byte(idx + 1), Data: resp.ShareData},
                node:  idx,
            }
        }(i, client)
    }

    var shares []shamir.Share
    var failures int
    for i := 0; i < 3; i++ {
        select {
        case r := <-results:
            shares = append(shares, r.share)
            if len(shares) == 2 {
                cancel() // Cancel remaining
                return shares, nil
            }
        case <-errs:
            failures++
            if failures > 1 {
                return nil, fmt.Errorf("insufficient shares: %d nodes failed", failures)
            }
        }
    }
    return nil, fmt.Errorf("insufficient shares: got %d, need 2", len(shares))
}
```

**Key detail:** Each returned share's `Data` field MUST be zeroized via `defer` in the calling Verify/Disable method. [VERIFIED: D-04 from CONTEXT.md]

### Pattern 2: Rate Limiting via Redis INCR+EXPIRE
**What:** Atomic increment with TTL for sliding window rate limiting.
**Redis commands:**

```go
// Source: go-redis INCR+EXPIRE pattern [VERIFIED: go-redis API]
func (rs *RedisStorage) IncrementRateLimit(ctx context.Context, key string, ttl time.Duration) (int64, error) {
    count, err := rs.client.Incr(ctx, key).Result()
    if err != nil {
        return 0, fmt.Errorf("incr rate limit: %w", err)
    }
    // Set TTL only on first increment (count == 1)
    if count == 1 {
        rs.client.Expire(ctx, key, ttl)
    }
    return count, nil
}
```

**Note on race condition:** Between INCR and EXPIRE there is a tiny window. Per D-05, this is acceptable — not using Lua scripts for simplicity. [VERIFIED: CONTEXT.md D-05]

### Pattern 3: Verify Flow Orchestration
**What:** Full OTP verification sequence.
**Order:**
1. Check rate limit (Redis INCR) -- fail if > 5
2. Retrieve 2 shares from MPC nodes (parallel)
3. Shamir Combine shares
4. Zeroize shares immediately
5. Check OTP reuse (Redis get counter)
6. ValidateOTPWithCounter(secret, code) -- get bool + counter
7. Zeroize combined secret immediately
8. If valid: store used counter in Redis, enable 2FA if first verification
9. Return result

### Pattern 4: Service Interface Extension for API Layer
**What:** The handler's `Service` interface in `twofa_service_api.go` needs Verify, Disable, GetStatus methods.

```go
// Source: existing pattern from twofa_service_api.go [VERIFIED: codebase]
type Service interface {
    Setup(ctx context.Context, userID, email string) (string, []string, error)
    Verify(ctx context.Context, userID, otpCode string) (bool, bool, error) // valid, isNewlyEnabled, error
    Disable(ctx context.Context, userID, otpCode string) error
    GetStatus(ctx context.Context, userID string) (*models.TwoFARecord, error)
}
```

### Anti-Patterns to Avoid
- **Logging share data or combined secrets:** SEC-05 violation. Only log user_id, operation, and status. [VERIFIED: CLAUDE.md]
- **Persisting the combined TOTP secret:** SEC-04 violation. Must be transient, zeroized after validation. [VERIFIED: CLAUDE.md]
- **Using errgroup for "first 2 wins":** errgroup waits for ALL goroutines. Use raw goroutines + channels for early cancellation. [VERIFIED: errgroup semantics]
- **Blocking on rate limit Redis failure:** D-07 says proceed without rate check on Redis unavailability. [VERIFIED: CONTEXT.md]

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Rate limit counter | Custom counter logic | Redis INCR + EXPIRE | Atomic, distributed, auto-expiry |
| Mock generation | Manual mock structs | minimock `//go:generate` | Consistent with existing codebase |
| Time counter for TOTP | Manual unix/30 math in service | ValidateOTPWithCounter in totp package | Keep crypto logic in crypto package |
| Parallel MPC calls | Sequential retry loop | goroutines + channels with context cancellation | Latency — parallel is 1x, sequential is 2-3x |

## Common Pitfalls

### Pitfall 1: ValidateOTP Does Not Return Counter
**What goes wrong:** Current `ValidateOTP` returns only `bool`. OTP reuse prevention (D-09) needs the matched counter to store in Redis.
**Why it happens:** Phase 5 implemented validation before reuse prevention was specified.
**How to avoid:** Add `ValidateOTPWithCounter(secret []byte, code string) (bool, int64)` that returns `(valid, matchedCounter)`. The counter is `uint64(time.Now().Unix()) / 30` adjusted for which window matched (T-1, T, T+1). Keep existing ValidateOTP for backward compatibility.
**Warning signs:** Tests pass but OTP reuse is not actually detected.

### Pitfall 2: Goroutine Leak in "First 2 Wins" Pattern
**What goes wrong:** The 3rd goroutine continues running after 2 results collected.
**Why it happens:** Forgetting to cancel context or goroutines blocking on channel send.
**How to avoid:** Use buffered channels (size 3) so goroutines never block on send. Cancel parent context after collecting 2 results. Each goroutine uses per-call context derived from the cancellable parent.
**Warning signs:** Test with `-race` flag shows goroutine leak warnings.

### Pitfall 3: Zeroize Before Use
**What goes wrong:** Defer zeroize runs before the secret is actually used for validation.
**Why it happens:** Putting `defer crypto.Zeroize(secret)` before `totp.ValidateOTPWithCounter(secret, code)` in the wrong scope.
**How to avoid:** Zeroize defers must be in the correct function scope. The combined secret must be zeroized AFTER ValidateOTPWithCounter call returns, not before. Place defer immediately after Combine returns, but ensure validation happens in the same function scope.
**Warning signs:** ValidateOTP always returns false because secret is already zeroed.

### Pitfall 4: Share Index Mismatch in RetrieveShare
**What goes wrong:** RetrieveShare request uses 0-based index but Shamir shares use 1-based.
**Why it happens:** MPC nodes store shares with the original 1-based index from Shamir Split.
**How to avoid:** Use `int32(i + 1)` for ShareIndex in RetrieveShareRequest, matching StoreShare. The Share struct returned must also use `byte(i + 1)` for Index.
**Warning signs:** Shamir Combine returns garbage because indices are wrong.

### Pitfall 5: Disable Cleanup Ordering
**What goes wrong:** If twofa_record is deleted first but share deletion fails, shares become orphans.
**Why it happens:** Wrong cleanup order.
**How to avoid:** Per D-12: delete shares FIRST, then backup_codes, then twofa_record. If share deletion fails, abort — twofa_record stays with is_enabled=true. User retries.
**Warning signs:** Orphaned encrypted shares in MPC nodes with no twofa_record to reference.

### Pitfall 6: Redis Key Format Inconsistency
**What goes wrong:** Rate limit key and OTP counter key use different user_id formats.
**Why it happens:** Different parts of code format keys differently.
**How to avoid:** Define key format constants in the service layer. Pass formatted keys to SessionStorage, not raw user_id.
**Warning signs:** Rate limiting doesn't work because keys don't match between increment and check.

## Code Examples

### ValidateOTPWithCounter (new function needed in totp package)
```go
// Source: extending existing totp.go pattern [VERIFIED: codebase totp.go]
// ValidateOTPWithCounter checks if code is valid for the current time +-1 window
// and returns the matched counter value for reuse prevention.
// Returns (false, 0) if no window matches.
func ValidateOTPWithCounter(secret []byte, code string) (bool, int64) {
    return validateOTPWithCounterAt(secret, code, time.Now().Unix())
}

func validateOTPWithCounterAt(secret []byte, code string, unixTime int64) (bool, int64) {
    if len(code) != 6 {
        return false, 0
    }
    for _, c := range code {
        if c < '0' || c > '9' {
            return false, 0
        }
    }

    counter := int64(unixTime) / 30

    if subtle.ConstantTimeCompare([]byte(hotp(secret, uint64(counter))), []byte(code)) == 1 {
        return true, counter
    }
    if subtle.ConstantTimeCompare([]byte(hotp(secret, uint64(counter+1))), []byte(code)) == 1 {
        return true, counter + 1
    }
    if counter > 0 {
        if subtle.ConstantTimeCompare([]byte(hotp(secret, uint64(counter-1))), []byte(code)) == 1 {
            return true, counter - 1
        }
    }
    return false, 0
}
```

### Redis Rate Limit Implementation
```go
// Source: go-redis pattern [VERIFIED: go-redis API]
func (rs *RedisStorage) IncrementRateLimit(ctx context.Context, key string, ttl time.Duration) (int64, error) {
    count, err := rs.client.Incr(ctx, key).Result()
    if err != nil {
        return 0, fmt.Errorf("incr rate limit: %w", err)
    }
    if count == 1 {
        rs.client.Expire(ctx, key, ttl)
    }
    return count, nil
}

func (rs *RedisStorage) GetRateLimit(ctx context.Context, key string) (int64, error) {
    count, err := rs.client.Get(ctx, key).Int64()
    if err == redis.Nil {
        return 0, nil
    }
    if err != nil {
        return 0, fmt.Errorf("get rate limit: %w", err)
    }
    return count, nil
}
```

### Redis OTP Counter Storage
```go
// Source: go-redis pattern [VERIFIED: go-redis API]
func (rs *RedisStorage) SetUsedOTPCounter(ctx context.Context, userID string, counter int64, ttl time.Duration) error {
    key := fmt.Sprintf("otp_used:%s", userID)
    return rs.client.Set(ctx, key, counter, ttl).Err()
}

func (rs *RedisStorage) GetUsedOTPCounter(ctx context.Context, userID string) (int64, error) {
    key := fmt.Sprintf("otp_used:%s", userID)
    val, err := rs.client.Get(ctx, key).Int64()
    if err == redis.Nil {
        return 0, nil
    }
    if err != nil {
        return 0, fmt.Errorf("get used otp counter: %w", err)
    }
    return val, nil
}
```

### PostgreSQL EnableTwoFA
```go
// Source: existing pgstorage pattern [VERIFIED: codebase pgstorage/twofa_record.go]
func (ps *PGStorage) EnableTwoFA(ctx context.Context, userID string) error {
    query := `UPDATE twofa_records SET is_enabled = TRUE WHERE user_id = $1`
    _, err := ps.pool.Exec(ctx, query, userID)
    if err != nil {
        return fmt.Errorf("enable twofa: %w", err)
    }
    return nil
}

func (ps *PGStorage) DeleteTwoFARecord(ctx context.Context, userID string) error {
    query := `DELETE FROM twofa_records WHERE user_id = $1`
    _, err := ps.pool.Exec(ctx, query, userID)
    if err != nil {
        return fmt.Errorf("delete twofa record: %w", err)
    }
    return nil
}

func (ps *PGStorage) DeleteBackupCodes(ctx context.Context, userID string) error {
    query := `DELETE FROM backup_codes WHERE user_id = $1`
    _, err := ps.pool.Exec(ctx, query, userID)
    if err != nil {
        return fmt.Errorf("delete backup codes: %w", err)
    }
    return nil
}
```

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + minimock v3 + gotest.tools/v3/assert |
| Config file | none (standard go test) |
| Quick run command | `cd twofa && go test ./internal/services/twofaService/ -run TestVerify -v -count=1` |
| Full suite command | `cd twofa && go test ./... -v -count=1 -race` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| 2FA-03 | Verify OTP: retrieve 2 shares, combine, validate, zeroize | unit | `go test ./internal/services/twofaService/ -run TestVerify_Success -v` | Wave 0 |
| 2FA-04 | First verify enables 2FA | unit | `go test ./internal/services/twofaService/ -run TestVerify_EnablesOnFirst -v` | Wave 0 |
| 2FA-05 | Rate limit 5 attempts/5min | unit | `go test ./internal/services/twofaService/ -run TestVerify_RateLimit -v` | Wave 0 |
| 2FA-06 | Disable 2FA: verify + delete shares + cleanup | unit | `go test ./internal/services/twofaService/ -run TestDisable -v` | Wave 0 |
| 2FA-07 | Check 2FA status | unit | `go test ./internal/services/twofaService/ -run TestGetStatus -v` | Wave 0 |
| 2FA-09 | OTP reuse rejection | unit | `go test ./internal/services/twofaService/ -run TestVerify_OTPReuse -v` | Wave 0 |
| SEC-05 | No share/key data in logs | unit | `go test ./internal/services/twofaService/ -run TestVerify_Zeroization -v` | Wave 0 |

### Sampling Rate
- **Per task commit:** `cd twofa && go test ./internal/services/twofaService/ -v -count=1 -race`
- **Per wave merge:** `cd twofa && go test ./... -v -count=1 -race`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `twofa/internal/services/twofaService/verify_test.go` -- covers 2FA-03, 2FA-04, 2FA-05, 2FA-09, SEC-05
- [ ] `twofa/internal/services/twofaService/disable_test.go` -- covers 2FA-06
- [ ] `twofa/internal/services/twofaService/status_test.go` -- covers 2FA-07
- [ ] `twofa/internal/services/twofaService/mocks/session_storage_mock.go` -- regenerate with new methods
- [ ] `twofa/internal/crypto/totp/totp_test.go` -- add tests for ValidateOTPWithCounter

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | OTP verification as second factor |
| V3 Session Management | no | Not directly (Redis used for rate limits, not sessions) |
| V4 Access Control | yes | 2FA status check requires user_id matching |
| V5 Input Validation | yes | OTP code format (6 digits), user_id format (UUID) |
| V6 Cryptography | yes | Shamir combine, TOTP validation -- all from existing packages, never hand-rolled here |

### Known Threat Patterns for This Phase

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| OTP brute force (10^6 codes) | Spoofing | Rate limit 5/5min per user (D-05, D-06) |
| OTP replay/reuse | Replay | Store last used counter in Redis, reject same counter (D-09, D-10) |
| Secret leakage via logs | Info Disclosure | Never log share data or combined secret (SEC-05), defer zeroize (D-04) |
| Timing side-channel on OTP comparison | Info Disclosure | subtle.ConstantTimeCompare already used in ValidateOTP [VERIFIED: codebase] |
| Share data exposure in error messages | Info Disclosure | Sanitized gRPC errors, no internal state leaked (SEC-02) |
| Denial of service via rate limit bypass | Denial of Service | Count ALL attempts not just failures (D-06), accept risk when Redis down (D-07) |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The "first 2 wins" pattern with raw goroutines + channels is the correct approach (vs errgroup) | Architecture Patterns | Medium -- errgroup could work with a different cancellation strategy, but it waits for all goroutines which adds latency |
| A2 | go-redis INCR returns 1 on first call to a non-existent key (auto-create behavior) | Code Examples | Low -- this is standard Redis INCR semantics, well-documented |
| A3 | `redis.Nil` is the sentinel error for key-not-found in go-redis v9 | Code Examples | Low -- standard go-redis pattern |

## Open Questions

1. **ValidateOTPWithCounter return type for counter**
   - What we know: Counter is `unix_time / 30`, can be very large. int64 is sufficient.
   - What's unclear: Should the function return 0 on failure, or should we use a pointer `*int64`?
   - Recommendation: Return `(bool, int64)` with counter=0 on failure. Simple, matches D-10 which only checks equality.

2. **Key formatting responsibility**
   - What we know: D-05 specifies key format `rate_limit:verify:{user_id}` and D-09 specifies `otp_used:{user_id}`
   - What's unclear: Should key formatting live in the service layer or in RedisStorage methods?
   - Recommendation: Service layer formats the full key string for rate limit (since the key pattern includes operation type). RedisStorage methods for OTP counter accept userID and format internally (since the key pattern is fixed).

3. **Disable2FA: should we re-verify via full Verify flow or inline?**
   - What we know: D-12 says "Verify OTP code" as step 1 of disable
   - What's unclear: Call `s.Verify()` internally or duplicate the share-retrieve-combine-validate logic?
   - Recommendation: Extract shared helper `verifyOTPInternal` used by both Verify and Disable. Avoids side effects of the full Verify (like enabling 2FA on first verify) being triggered during disable.

## Sources

### Primary (HIGH confidence)
- Codebase inspection: `twofa/internal/services/twofaService/twofa_service.go` -- interfaces, service struct
- Codebase inspection: `twofa/internal/services/twofaService/setup.go` -- errgroup pattern, zeroize pattern
- Codebase inspection: `twofa/internal/crypto/totp/totp.go` -- ValidateOTP signature (bool only, no counter)
- Codebase inspection: `twofa/internal/crypto/shamir/shamir.go` -- Combine API, Share struct
- Codebase inspection: `twofa/internal/storage/redisstorage/redisstorage.go` -- RedisStorage struct
- Codebase inspection: `twofa/internal/storage/pgstorage/twofa_record.go` -- existing methods
- Codebase inspection: `twofa/api/twofa_api/twofa_service.proto` -- Verify2FA, Disable2FA, Get2FAStatus messages
- Codebase inspection: `twofa/internal/services/twofaService/setup_test.go` -- test patterns with minimock

### Secondary (MEDIUM confidence)
- Phase 7 CONTEXT.md -- established patterns for MPC coordination, zeroize, errgroup
- Phase 8 CONTEXT.md -- all locked decisions (D-01 through D-17)

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all libraries already in use, no new dependencies
- Architecture: HIGH -- follows established patterns from Phase 7, only "first 2 wins" is novel
- Pitfalls: HIGH -- identified from codebase inspection (ValidateOTP missing counter is concrete, not speculative)
- Security: HIGH -- zeroize pattern established, constant-time comparison already in place

**Research date:** 2026-04-12
**Valid until:** 2026-05-12 (stable -- internal project, no external API changes expected)
