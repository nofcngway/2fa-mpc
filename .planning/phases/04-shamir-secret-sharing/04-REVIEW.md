---
phase: 04-shamir-secret-sharing
reviewed: 2026-04-12T12:00:00Z
depth: standard
files_reviewed: 4
files_reviewed_list:
  - twofa/internal/crypto/shamir/gf256.go
  - twofa/internal/crypto/shamir/gf256_test.go
  - twofa/internal/crypto/shamir/shamir.go
  - twofa/internal/crypto/shamir/shamir_test.go
findings:
  critical: 1
  warning: 1
  info: 0
  total: 2
status: issues_found
---

# Phase 4: Code Review Report

**Reviewed:** 2026-04-12T12:00:00Z
**Depth:** standard
**Files Reviewed:** 4
**Status:** issues_found

## Summary

The Shamir Secret Sharing implementation over GF(256) is well-structured and mathematically correct. GF(256) arithmetic uses standard log/exp table lookup with the AES irreducible polynomial, and all field properties (commutativity, associativity, distributivity, inverses) are exhaustively tested. The Split/Combine roundtrip works correctly for the 2-of-3 scheme. Input validation covers most edge cases. Two issues were found: one critical security gap (secret material not zeroed from memory after use) and one missing validation that could cause silent incorrect reconstruction.

## Critical Issues

### CR-01: Secret-bearing `coeffs` buffer not zeroed after use

**File:** `twofa/internal/crypto/shamir/shamir.go:89-102`
**Issue:** The `coeffs` slice holds the secret byte at `coeffs[0]` during each iteration of the split loop. After `Split` returns, this buffer remains in memory containing the last secret byte and its random coefficients. The project security rules (CLAUDE.md) explicitly require: "After split/combine in memory, immediately zeroize using subtle.ConstantTimeCompare or manual byte clearing." This leaves secret material recoverable from process memory via heap inspection or core dumps.
**Fix:**
```go
// Add at the end of Split, before the return statement (after line 102):
// Zeroize coefficient buffer — contains secret bytes.
for i := range coeffs {
    coeffs[i] = 0
}
```

Note: `coeffs[0]` only holds the *last* secret byte at function exit, but all secret bytes passed through this buffer during the loop. A determined attacker with memory access could potentially recover partial secret data from heap artifacts. The zeroization should be done with a `defer` to ensure it runs even if `io.ReadFull` returns an error mid-way:

```go
coeffs := make([]byte, threshold)
defer func() {
    for i := range coeffs {
        coeffs[i] = 0
    }
}()
```

## Warnings

### WR-01: `Combine` does not reject shares with `Index: 0`

**File:** `twofa/internal/crypto/shamir/shamir.go:111-157`
**Issue:** A share with `Index: 0` is never validated against. In this scheme, x=0 is the evaluation point that yields the secret itself. If a share with `Index: 0` is passed to `Combine`, the Lagrange interpolation computes its basis polynomial as 1 (since for that share, every term in the product becomes `gfDiv(xs[j], gfAdd(0, xs[j])) = gfDiv(xs[j], xs[j]) = 1`). This means the y-value at x=0 would be added directly to the result, producing a silently incorrect reconstruction unless the share data happens to equal the secret. While `Split` never generates Index=0 shares, `Combine` accepts arbitrary `Share` structs, so a caller could pass fabricated or corrupted shares with zero index.
**Fix:**
```go
// Add to the validation section in Combine, after the duplicate index check (after line 138):
for _, s := range shares {
    if s.Index == 0 {
        return nil, errors.New("shamir: share index must not be zero (x=0 is reserved for the secret)")
    }
}
```

Or define a sentinel error alongside the others:
```go
var ErrZeroIndex = errors.New("shamir: share index must not be zero")
```

---

_Reviewed: 2026-04-12T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
