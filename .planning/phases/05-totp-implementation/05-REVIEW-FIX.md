---
phase: 05-totp-implementation
fixed_at: 2026-04-12
status: all_fixed
findings_in_scope: 5
fixed: 5
skipped: 0
iteration: 1
---

# Phase 05: Code Review Fix Report

## Fixes Applied

### CR-01: Timing side-channel in OTP validation (FIXED)

**File:** `twofa/internal/crypto/totp/totp.go`
**Fix:** Replaced `==` string comparison with `crypto/subtle.ConstantTimeCompare` in `validateOTPAt`. All three time window comparisons now use constant-time comparison.

### WR-01: Unsigned integer underflow on counter subtraction (FIXED)

**File:** `twofa/internal/crypto/totp/totp.go`
**Fix:** Replaced loop over `[]uint64{counter - 1, counter, counter + 1}` with explicit checks: current and next window always checked, previous window only checked when `counter > 0`. Added `TestValidateOTP_CounterZero` test.

### WR-02: Secret parameter not URL-encoded in provisioning URI (FIXED)

**File:** `twofa/internal/crypto/totp/uri.go`
**Fix:** Added `url.QueryEscape(secret)` to encode the secret parameter in the query string.

### IN-01: No input validation for nil/empty secret (FIXED)

**File:** `twofa/internal/crypto/totp/totp.go`
**Fix:** Added `len(secret) == 0` check at the top of `hotp()` that panics — indicates a bug in the caller. Added `TestHOTP_EmptySecretPanics` test.

### IN-02: ValidateOTP does not explicitly reject non-numeric codes (FIXED)

**File:** `twofa/internal/crypto/totp/totp.go`
**Fix:** Added digit-only validation loop after length check in `validateOTPAt`: rejects any code containing non-ASCII-digit characters.

## Additional Changes

- **Test framework migration:** Rewrote all tests to use `gotest.tools/v3/assert` with suite pattern (`totpSuite` struct + `newTOTPSuite()` helper), matching the auth service test conventions.
- **Makefile:** Added `mock` target (empty — no interfaces to mock in totp package yet).
- **Test count:** 19 tests (17 original + 2 new), all passing.

## Verification

```
go test ./internal/crypto/totp/ -v -count=1  → 19/19 PASS
go vet ./internal/crypto/totp/               → clean
go test ./internal/crypto/... -count=1       → shamir + totp both PASS
```
