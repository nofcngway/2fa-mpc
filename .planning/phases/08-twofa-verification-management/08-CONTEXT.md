# Phase 8: TwoFA Verification & Management - Context

**Gathered:** 2026-04-12
**Status:** Ready for planning

<domain>
## Phase Boundary

Implement OTP verification (retrieve 2 shares from MPC nodes, Shamir combine, TOTP validate), 2FA enable on first verify, rate limiting (5 attempts/5min), OTP reuse prevention, disable 2FA (verify + delete shares + cleanup), and 2FA status check. All crypto primitives (Shamir, TOTP), MPC node service, and setup flow already exist from Phases 4-7.

</domain>

<decisions>
## Implementation Decisions

### Share Retrieval Strategy
- **D-01:** Query all 3 MPC nodes in parallel via errgroup (same pattern as Phase 7 setup). Use first 2 successful responses for Shamir combine. Cancel remaining on 2 successes. Tolerates 1 node being down.
- **D-02:** If only 1 or 0 nodes respond successfully, return `codes.Internal` error. Minimum 2 shares required for 2-of-3 threshold.
- **D-03:** When 2 shares succeed but 3rd fails, proceed silently. Log node failure internally via slog for ops visibility. No warning to caller — 2-of-3 is the design.
- **D-04:** `defer zeroize()` for each retrieved share's Data field AND the combined secret, immediately after retrieval/combine. Same pattern as Phase 7 setup (D-08, D-09, D-10). Use shared zeroize utility from `twofa/internal/crypto/`.

### Rate Limiting
- **D-05:** Redis `INCR` + `EXPIRE` pattern. Key: `rate_limit:verify:{user_id}`. EXPIRE 300s (5 minutes) set on first increment. Max 5 attempts.
- **D-06:** Count ALL verification attempts regardless of outcome (not just failures). Prevents brute force even with lucky guesses.
- **D-07:** When Redis is unavailable, log warning and proceed without rate check. Availability over strict enforcement — rate limiting is defense-in-depth, not primary security.
- **D-08:** SessionStorage interface gets methods: `IncrementRateLimit(ctx, key string, ttl time.Duration) (int64, error)` and `GetRateLimit(ctx, key string) (int64, error)`.

### OTP Reuse Prevention
- **D-09:** Store last validated TOTP time counter in Redis. Key: `otp_used:{user_id}`, value: time counter (int64), TTL: 90s (covers 3 TOTP windows).
- **D-10:** Reject if new counter == stored counter (exact match). Different time windows (t-1, t, t+1) have different counter values, so reuse across adjacent windows is allowed — only same-counter reuse blocked.
- **D-11:** SessionStorage interface gets methods: `SetUsedOTPCounter(ctx, userID string, counter int64, ttl time.Duration) error` and `GetUsedOTPCounter(ctx, userID string) (int64, error)`.

### Disable 2FA Cleanup
- **D-12:** Disable2FA order: 1) Verify OTP code, 2) Delete shares from all 3 MPC nodes in parallel (errgroup), 3) Delete backup codes from PostgreSQL, 4) Delete twofa_record from PostgreSQL. Shares are sensitive data — removed first.
- **D-13:** If share deletion fails on any MPC node, return `codes.Internal`. twofa_record stays (is_enabled=true). User retries disable. No orphaned metadata.
- **D-14:** On successful disable, clean up Redis keys: DEL `rate_limit:verify:{user_id}` and `otp_used:{user_id}`. Clean slate if user re-enables 2FA later.
- **D-15:** SessionStorage gets method: `DeleteKeys(ctx context.Context, keys ...string) error` for cleanup.

### Storage Interface Extensions
- **D-16:** Storage interface gets new methods for Phase 8:
  - `EnableTwoFA(ctx, userID string) error` — UPDATE twofa_records SET is_enabled=true
  - `DeleteTwoFARecord(ctx, userID string) error` — DELETE from twofa_records
  - `DeleteBackupCodes(ctx, userID string) error` — DELETE from backup_codes
- **D-17:** Get2FAStatus handler queries `GetTwoFARecord` (already exists) and returns is_enabled + created_at. No new storage method needed.

### Claude's Discretion
- Exact errgroup cancellation pattern for "first 2 wins" retrieval
- Whether to wrap rate limit check + OTP validation in a single service method or separate
- Prometheus metric labels for verify/disable/status operations
- Kafka audit event structure for `2fa.verified`, `2fa.disabled`, `2fa.status_checked`
- Internal helper decomposition within Verify and Disable methods
- Error message wording (must not leak internal state per SEC-02)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Architecture & Structure
- `CLAUDE.md` — Full project spec, TOTP never persisted, Shamir 2-of-3, zeroization rules, no secret logging, rate limiting spec
- `workspace/02 - Services/TwoFA Service.md` — TwoFA API, orchestration flow, MPC coordination

### Requirements
- `.planning/REQUIREMENTS.md` — 2FA-03, 2FA-04, 2FA-05, 2FA-06, 2FA-07, 2FA-09, SEC-05

### Phase 7 Context (direct dependency)
- `.planning/phases/07-twofa-setup-flow/07-CONTEXT.md` — MPC communication patterns (errgroup, compensating delete), zeroize utility, backup code format, is_enabled lifecycle

### Crypto Packages (Phases 4-5)
- `twofa/internal/crypto/shamir/shamir.go` — `Combine(shares) ([]byte, error)`
- `twofa/internal/crypto/totp/totp.go` — `ValidateOTP(secret []byte, code string) bool` with +-1 window

### MPC Node Service (Phase 6)
- `mpc/api/mpc_api/mpc.proto` — RetrieveShare, DeleteShare RPC definitions
- `.planning/phases/06-mpc-node-service/06-CONTEXT.md` — RetrieveShare returns NotFound, DeleteShare idempotent

### Existing TwoFA Code (Phases 1, 7)
- `twofa/internal/services/twofaService/twofa_service.go` — Storage/SessionStorage/MPCClient interfaces, TwoFAService struct
- `twofa/internal/api/twofa_service_api/verify.go` — Verify2FA handler stub
- `twofa/internal/api/twofa_service_api/disable.go` — Disable2FA handler stub
- `twofa/internal/api/twofa_service_api/status.go` — Get2FAStatus handler stub
- `twofa/internal/storage/redisstorage/redisstorage.go` — RedisStorage with client (no rate limit methods yet)
- `twofa/internal/storage/pgstorage/twofa_record.go` — GetTwoFARecord already implemented
- `twofa/internal/models/models.go` — TwoFARecord (UserID, IsEnabled, CreatedAt), BackupCode models

### Proto Definitions
- `twofa/api/twofa_api/twofa_service.proto` — Verify2FA, Disable2FA, Get2FAStatus messages defined

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `twofa/internal/crypto/shamir/` — Combine fully implemented and tested (Phase 4)
- `twofa/internal/crypto/totp/` — ValidateOTP with +-1 window implemented (Phase 5)
- `twofa/internal/crypto/zeroize.go` — Zeroize utility shared between setup and verify (Phase 7 D-10)
- `twofa/internal/services/twofaService/setup.go` — errgroup pattern for parallel MPC calls, compensating delete pattern
- `twofa/internal/services/twofaService/mocks/` — minimock-generated mocks for Storage, SessionStorage, MPCClient
- `twofa/internal/storage/pgstorage/` — PGStorage with twofa_records + backup_codes tables, GetTwoFARecord method

### Established Patterns
- One file per gRPC method in handler directory (verify.go, disable.go, status.go stubs exist)
- One file per service method in service directory (setup.go exists as reference)
- Interface-based DI with minimock for mock generation
- errgroup with shared context for parallel MPC calls (Phase 7 setup.go)
- `defer zeroize()` immediately after secret/share allocation

### Integration Points
- SessionStorage interface needs rate limit + OTP reuse methods (currently empty)
- RedisStorage needs implementations for rate limiting and OTP counter operations
- Storage interface needs EnableTwoFA, DeleteTwoFARecord, DeleteBackupCodes methods
- PGStorage needs these new SQL implementations
- Service needs Verify, Disable, GetStatus methods
- Handlers need request validation + service delegation + gRPC status code mapping

</code_context>

<specifics>
## Specific Ideas

- "First 2 wins" parallel retrieval — errgroup queries all 3 MPC nodes, context cancelled after 2 successes
- Rate limit key `rate_limit:verify:{user_id}` with INCR+EXPIRE, not Lua script — simpler, acceptable race
- OTP reuse via Redis time counter comparison, not full window tracking — lightweight and sufficient
- Disable cleanup deletes shares BEFORE metadata — prioritize removing sensitive data
- Redis key cleanup on disable — clean slate for potential re-enrollment

</specifics>

<deferred>
## Deferred Ideas

- **Kafka audit events** — Phase 9 adds `2fa.verified`, `2fa.disabled`, `2fa.status_checked` events
- **Prometheus metrics** — Phase 9 adds verify/disable/status operation counters and latency histograms
- **Backup code verification** — Could be added as alternative to OTP in verify flow, but not in current phase scope

None — discussion stayed within phase scope

</deferred>

---

*Phase: 08-twofa-verification-management*
*Context gathered: 2026-04-12*
