---
phase: 05-totp-implementation
reviewed: 2026-04-12T12:00:00Z
depth: standard
files_reviewed: 3
files_reviewed_list:
  - twofa/internal/crypto/totp/totp.go
  - twofa/internal/crypto/totp/totp_test.go
  - twofa/internal/crypto/totp/uri.go
findings:
  critical: 1
  warning: 2
  info: 2
  total: 5
status: issues_found
---

# Phase 5: Code Review Report

**Reviewed:** 2026-04-12T12:00:00Z
**Depth:** standard
**Files Reviewed:** 3
**Status:** issues_found

## Summary

Reviewed the TOTP implementation consisting of OTP generation/validation (totp.go), provisioning URI generation (uri.go), and comprehensive tests (totp_test.go). The core HOTP/TOTP algorithm correctly implements RFC 4226/6238 and passes all RFC test vectors. The test suite is thorough with good coverage of edge cases, boundary conditions, and validation scenarios.

Key concerns: (1) OTP comparison uses standard string equality which is vulnerable to timing side-channel attacks -- this is the most significant security issue. (2) Counter underflow at very low timestamps. (3) Secret parameter not URL-encoded in provisioning URI.

## Critical Issues

### CR-01: Timing side-channel in OTP validation

**File:** `twofa/internal/crypto/totp/totp.go:49`
**Issue:** The OTP comparison `hotp(secret, c) == code` uses Go's standard string equality operator, which short-circuits on the first mismatched byte. An attacker performing online brute-force can measure response time differences to determine how many leading characters of the OTP match, significantly reducing the search space from 1,000,000 to roughly 6*10 = 60 attempts in the worst theoretical case. For a security-critical 2FA verification endpoint, constant-time comparison is required.
**Fix:**
```go
import "crypto/subtle"

func validateOTPAt(secret []byte, code string, unixTime int64) bool {
	if len(code) != 6 {
		return false
	}

	counter := uint64(unixTime) / 30
	for _, c := range []uint64{counter - 1, counter, counter + 1} {
		expected := hotp(secret, c)
		if subtle.ConstantTimeCompare([]byte(expected), []byte(code)) == 1 {
			return true
		}
	}
	return false
}
```

## Warnings

### WR-01: Unsigned integer underflow on counter subtraction

**File:** `twofa/internal/crypto/totp/totp.go:48`
**Issue:** When `unixTime` is in the range [0, 29], `counter` is 0 and the expression `counter - 1` underflows to `uint64(math.MaxUint64)`. This produces an HOTP value for an astronomically large counter, which is incorrect behavior. While unlikely in production (Unix timestamps are well past 0), this is a logic error that violates the contract of "check T-1 window." It could also manifest if a caller passes a negative time that gets cast to a small unsigned value.
**Fix:**
```go
func validateOTPAt(secret []byte, code string, unixTime int64) bool {
	if len(code) != 6 {
		return false
	}

	counter := uint64(unixTime) / 30
	// Check current and next window always; check previous only if counter > 0.
	if subtle.ConstantTimeCompare([]byte(hotp(secret, counter)), []byte(code)) == 1 {
		return true
	}
	if subtle.ConstantTimeCompare([]byte(hotp(secret, counter+1)), []byte(code)) == 1 {
		return true
	}
	if counter > 0 {
		if subtle.ConstantTimeCompare([]byte(hotp(secret, counter-1)), []byte(code)) == 1 {
			return true
		}
	}
	return false
}
```

### WR-02: Secret parameter not URL-encoded in provisioning URI

**File:** `twofa/internal/crypto/totp/uri.go:13`
**Issue:** The `secret` string is interpolated directly into the query string without `url.QueryEscape()`. While standard base32 characters (A-Z, 2-7) are URL-safe, this assumes the caller always passes a correctly formatted base32 string. If a malformed or non-base32 secret is passed, the resulting URI could be malformed or contain injection into other query parameters (e.g., a secret containing `&issuer=evil`).
**Fix:**
```go
func GenerateProvisioningURI(secret string, email string) string {
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		url.PathEscape(issuer),
		url.PathEscape(email),
		url.QueryEscape(secret),
		url.QueryEscape(issuer),
	)
}
```

## Info

### IN-01: No input validation for nil/empty secret

**File:** `twofa/internal/crypto/totp/totp.go:31-38`
**Issue:** `GenerateOTP` and `ValidateOTP` do not validate that `secret` is non-nil and non-empty. Passing a nil or empty secret to `hmac.New(sha1.New, secret)` silently succeeds and produces a deterministic output for an empty key. This could mask bugs in calling code where the secret was not properly reconstructed from Shamir shares.
**Fix:** Add a guard at the top of `hotp()`:
```go
func hotp(secret []byte, counter uint64) string {
	if len(secret) == 0 {
		// This should never happen in production; indicates a bug in the caller.
		panic("totp: hotp called with empty secret")
	}
	// ... rest of function
}
```
Alternatively, return an error tuple from `GenerateOTP` and `ValidateOTP` instead of panicking.

### IN-02: ValidateOTP does not explicitly reject non-numeric codes

**File:** `twofa/internal/crypto/totp/totp.go:42-54`
**Issue:** `validateOTPAt` checks that the code is exactly 6 characters but does not verify that all characters are ASCII digits. Non-numeric codes like "12345a" are rejected only because they fail to match any HOTP output, not because of explicit validation. Adding a digit check would provide clearer error semantics and faster rejection.
**Fix:**
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
	// ... rest of validation
}
```

---

_Reviewed: 2026-04-12T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
