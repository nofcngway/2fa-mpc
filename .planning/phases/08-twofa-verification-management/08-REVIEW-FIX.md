---
phase: 08-twofa-verification-management
fixed_at: 2026-04-12T17:05:00Z
review_path: .planning/phases/08-twofa-verification-management/08-REVIEW.md
iteration: 1
findings_in_scope: 4
fixed: 4
skipped: 0
status: all_fixed
---

# Phase 08: Code Review Fix Report

**Fixed at:** 2026-04-12T17:05:00Z
**Source review:** .planning/phases/08-twofa-verification-management/08-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 4
- Fixed: 4
- Skipped: 0

## Fixed Issues

### CR-01: Rate-limit TTL silently ignored -- user can be permanently locked out

**Files modified:** `twofa/internal/storage/redisstorage/rate_limit.go`
**Commit:** 793affc
**Applied fix:** Added error checking on the `rs.client.Expire()` call in `IncrementRateLimit`. When `Expire` fails, the key is deleted via `Del` as a best-effort fallback to prevent an immortal counter, and the error is returned to the caller. This ensures a failed TTL set does not leave a permanent rate-limit key that locks the user out of 2FA verification.

### WR-01: Disable path skips OTP reuse prevention -- captured code can be replayed

**Files modified:** `twofa/internal/services/twofaService/disable.go`, `twofa/internal/services/twofaService/disable_test.go`
**Commits:** 17f7a5d, 53918be
**Applied fix:** The code already used `ValidateOTPWithCounter` (contrary to the review's description of `ValidateOTP`), but the matched counter was discarded with `_`. Changed to capture `matchedCounter`, added OTP reuse check via `GetUsedOTPCounter` (same pattern as `Verify`), and added `SetUsedOTPCounter` to store the used counter. Updated `TestDisable_Success` and `TestDisable_ShareDeletionFails` tests to set mock expectations for the new `GetUsedOTPCounter` and `SetUsedOTPCounter` calls.

### WR-02: OTP reuse check exempts counter value 0 -- codes at epoch can be replayed

**Files modified:** `twofa/internal/services/twofaService/verify.go`
**Commit:** 56ed1c6
**Applied fix:** Removed the `lastCounter != 0` guard from the OTP reuse check condition in `Verify`. The condition was redundant because `hasLastCounter` already correctly distinguishes the "key not present" case from a genuine counter-0 match. The guard was a latent correctness bug: any TOTP code matching counter 0 could be replayed indefinitely.

### WR-03: `EnableTwoFA` silently ignores zero rows-affected

**Files modified:** `twofa/internal/storage/pgstorage/twofa_record.go`
**Commit:** e3f6738
**Applied fix:** Changed `EnableTwoFA` to capture the `CommandTag` from `pool.Exec` and check `tag.RowsAffected()`. If zero rows were affected (record deleted between `GetTwoFARecord` and `EnableTwoFA`), the method now returns an error instead of silently succeeding.

## Skipped Issues

None -- all findings were fixed.

---

_Fixed: 2026-04-12T17:05:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
