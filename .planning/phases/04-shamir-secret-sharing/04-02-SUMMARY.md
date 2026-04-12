---
phase: 04-shamir-secret-sharing
plan: 02
subsystem: crypto/shamir
tags: [shamir, split, combine, lagrange, gf256, tdd]
dependency_graph:
  requires: [04-01]
  provides: [Split, Combine, Share]
  affects: [twofa/internal/crypto/shamir/]
tech_stack:
  added: []
  patterns: [horner-evaluation, lagrange-interpolation, sentinel-errors]
key_files:
  created:
    - twofa/internal/crypto/shamir/shamir.go
    - twofa/internal/crypto/shamir/shamir_test.go
  modified: []
decisions:
  - Used io.ReadFull(rand.Reader) for random coefficient generation ensuring exactly threshold-1 bytes per secret byte
  - Lagrange interpolation computes basis polynomials using GF(256) arithmetic where subtraction equals addition (XOR)
  - Validation order in Combine checks share count first, then empty data, then length mismatch, then duplicate indices
metrics:
  duration: ~2min
  completed: 2026-04-12T05:51:00Z
  tests_added: 16
  tests_total: 30
  files_created: 2
  files_modified: 0
requirements:
  - CRYPTO-01
  - CRYPTO-03
---

# Phase 4 Plan 02: Shamir Split/Combine Summary

Shamir Secret Sharing Split and Combine over GF(256) with Horner polynomial evaluation and Lagrange interpolation -- 2-of-3 threshold scheme with crypto/rand coefficients and comprehensive input validation via sentinel errors.

## What Was Built

### Share Type
- `Share` struct with `Index byte` (1-based x-coordinate) and `Data []byte` (y-values, same length as secret)

### Split Function
- Validates inputs: empty secret, threshold < 2, n < threshold, n > 255
- For each byte of the secret, constructs a random polynomial of degree (threshold-1) with the secret byte as constant term
- Random coefficients generated via `crypto/rand.Reader` (CSPRNG) -- fresh randomness per byte
- Evaluates polynomial at each share index using Horner's method in GF(256)
- Returns n shares with 1-based indices

### Combine Function
- Validates inputs: fewer than 2 shares, empty share data, mismatched data lengths, duplicate indices
- Reconstructs secret via Lagrange interpolation at x=0 in GF(256)
- Exploits GF(2^n) property: subtraction equals addition (XOR), simplifying basis polynomial computation

### Internal Helpers
- `evalPolynomial(coeffs, x)`: Horner's method evaluation in GF(256)
- `lagrangeInterpolateAtZero(xs, ys)`: Lagrange basis polynomial computation and accumulation

### Test Suite (16 tests)
- 4 roundtrip tests: 20-byte, 1-byte, 32-byte secrets with all share pair combinations, plus all-3-shares
- 3 security property tests: single share rejection, share distinctness, randomness across calls
- 4 Split validation tests: empty secret, threshold too low, n < threshold, too many shares
- 3 Combine validation tests: duplicate indices, empty share data, mismatched lengths
- 2 structure tests: 1-based indices, correct data length

## Commits

| Hash | Type | Description |
|------|------|-------------|
| 96b1f5f | test | Add 16 failing tests for Shamir Split/Combine (RED) |
| 04b5ea9 | feat | Implement Split/Combine with Lagrange interpolation (GREEN) |

## Deviations from Plan

None -- plan executed exactly as written.

## Threat Mitigations Applied

| Threat ID | Mitigation |
|-----------|------------|
| T-04-03 | `crypto/rand.Reader` used exclusively for polynomial coefficients; no `math/rand` anywhere |
| T-04-04 | Fresh `io.ReadFull(rand.Reader, coeffs[1:])` call per byte of secret -- no coefficient reuse |
| T-04-05 | All public functions return errors for invalid input; internal gfDiv/gfInv panics only reachable with validated data |
| T-04-06 | Combine validates unique indices via map before interpolation; returns ErrDuplicateIndex |

## Verification

```
go test ./internal/crypto/shamir/ -v -count=1 -race  # 30 PASS (14 gf256 + 16 shamir)
go vet ./internal/crypto/shamir/                       # clean
```

## Self-Check: PASSED
