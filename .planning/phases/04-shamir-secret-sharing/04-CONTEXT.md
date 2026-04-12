# Phase 4: Shamir Secret Sharing - Context

**Gathered:** 2026-04-12
**Status:** Ready for planning

<domain>
## Phase Boundary

Implement Shamir Secret Sharing from scratch: GF(256) finite field arithmetic (log/exp tables, add, mul, div, inverse) and Split/Combine functions with 2-of-3 threshold scheme. Pure cryptographic library — no service dependencies, no storage, no gRPC. Located in the TwoFA service module, consumed by TwoFA orchestration in Phase 7.

</domain>

<decisions>
## Implementation Decisions

### Package Location
- **D-01:** Package at `twofa/internal/crypto/shamir/` — NOT inside `twofaService/`. Crypto is a separate concern from business logic. TwoFA service imports it in Phase 7.
- **D-02:** Three files: `gf256.go` (field arithmetic), `shamir.go` (Split/Combine), `shamir_test.go` (tests). GF(256) tests can be in `gf256_test.go` if needed.

### API Design
- **D-03:** Package-level functions, NOT methods on a struct. Stateless API:
  - `Split(secret []byte, n, threshold int) ([]Share, error)`
  - `Combine(shares []Share) ([]byte, error)`
- **D-04:** `Share` struct: `Index byte` (x-coordinate: 1, 2, 3) + `Data []byte` (y-values, same length as secret)
- **D-05:** Share indices are 1-based (1, 2, 3). x=0 is reserved for the secret value f(0). Maps naturally to MPC node IDs.
- **D-06:** Input validation: Split returns error for empty secret, n < threshold, threshold < 2, n > 255. Combine returns error for insufficient shares, duplicate indices, empty shares.

### GF(256) Arithmetic
- **D-07:** Generator polynomial: 0x11B (x^8 + x^4 + x^3 + x + 1) — AES/Rijndael polynomial
- **D-08:** Log/exp tables generated at runtime in `init()` function — shows the algorithm construction, better for  demonstration than hardcoded magic numbers
- **D-09:** Operations: `gfAdd(a, b byte) byte` (XOR), `gfMul(a, b byte) byte` (via log/exp), `gfDiv(a, b byte) byte`, `gfInv(a byte) byte`
- **D-10:** Polynomial evaluation and Lagrange interpolation as internal helpers

### Test Strategy
- **D-11:** Maximum coverage (~25+ tests). This is a  — comprehensive testing demonstrates cryptographic correctness:
  - GF(256) arithmetic: add commutativity/associativity, mul identity/commutativity, div inverse, mul by zero, exp/log table consistency
  - Split→Combine roundtrip for all 3 combinations of 2-of-3
  - 1-of-3 does NOT recover secret
  - Edge cases: empty secret (error), 1-byte secret, 20-byte secret (TOTP size), 32-byte secret
  - Invalid inputs: n < threshold, threshold < 2, duplicate share indices, corrupted share data
  - Determinism: same secret + different random coefficients → different shares (statistical test)

### Claude's Discretion
- Exact internal helper function decomposition (polynomial evaluation, Lagrange basis)
- Error type design (sentinel errors vs custom types)
- Whether to use `crypto/rand` directly or accept an `io.Reader` for testability
- GF(256) function export level (export arithmetic for testing or keep internal)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Architecture & Structure
- `CLAUDE.md` — Shamir 2-of-3 requirement, GF(256), no third-party libraries
- `workspace/03 - Security/Shamir Secret Sharing.md` — Algorithm spec, polynomial, file structure, test cases

### Requirements
- `.planning/REQUIREMENTS.md` — CRYPTO-01, CRYPTO-02, CRYPTO-03

### Integration Points (Phase 7)
- `twofa/internal/services/twofaService/twofa_service.go` — Will import shamir package for Split/Combine

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- TwoFA Go module already exists (`twofa/go.mod`) — no module setup needed
- `twofa/internal/` directory structure established from Phase 1
- Test patterns from auth service (gotest.tools/v3/assert) available but may not be needed — standard `testing` sufficient for pure math

### Established Patterns
- One file per concern (established in auth service and bootstrap refactoring)
- Package-level functions for stateless operations
- `crypto/rand` for random byte generation (used in auth for UUIDs)

### Integration Points
- `twofa/internal/crypto/shamir/` — new package, no existing code to integrate with
- Phase 7 will import this package from `twofaService` for 2FA setup flow

</code_context>

<specifics>
## Specific Ideas

- Log/exp table generation in `init()` demonstrates understanding of GF(256) construction — important for  defense
- Share.Index as `byte` (not `int`) since GF(256) elements are 0-255
- Split processes each byte of the secret independently through the polynomial
- Lagrange interpolation at x=0 recovers the secret byte

</specifics>

<deferred>
## Deferred Ideas

- **TOTP integration** — Phase 5 implements TOTP, Phase 7 wires Shamir+TOTP together
- **Secret zeroization** — Phase 7 handles zeroizing reconstructed secrets after use
- **Share encryption** — Phase 6 (MPC Node) handles AES-256-GCM encryption at rest

</deferred>

---

*Phase: 04-shamir-secret-sharing*
*Context gathered: 2026-04-12*
