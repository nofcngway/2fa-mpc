# Phase 5: TOTP Implementation - Context

**Gathered:** 2026-04-12
**Status:** Ready for planning

<domain>
## Phase Boundary

Implement TOTP (Time-Based One-Time Password) per RFC 6238 from scratch: secret generation, OTP computation via HMAC-SHA1 with dynamic truncation, time-window validation (±1 period), and provisioning URI generation. Pure cryptographic library — no service dependencies, no storage, no gRPC. Located in the TwoFA service module, consumed by TwoFA orchestration in Phase 7.

</domain>

<decisions>
## Implementation Decisions

### Package Location
- **D-01:** Package at `twofa/internal/crypto/totp/` — consistent with Phase 4 pattern (`twofa/internal/crypto/shamir/`). Crypto is a separate concern from business logic.
- **D-02:** Two source files: `totp.go` (core logic: HMAC, truncation, GenerateSecret, GenerateOTP, ValidateOTP) and `uri.go` (GenerateProvisioningURI). Tests in `totp_test.go`.

### API Design
- **D-03:** Four exported package-level functions, stateless API:
  - `GenerateSecret() ([]byte, string, error)` — returns raw 20-byte secret + base32-encoded string
  - `GenerateOTP(secret []byte, time int64) string` — produces 6-digit code for given unix timestamp
  - `ValidateOTP(secret []byte, code string) bool` — checks code against current time ±1 window
  - `GenerateProvisioningURI(secret string, email string) string` — returns otpauth:// URI
- **D-04:** HOTP computation (HMAC-SHA1 + dynamic truncation) is an internal unexported helper, not part of public API.

### Provisioning URI
- **D-05:** Fixed issuer "MPC-2FA", label format `MPC-2FA:{email}`. URI format:
  `otpauth://totp/MPC-2FA:{email}?secret={base32}&issuer=MPC-2FA&algorithm=SHA1&digits=6&period=30`
- **D-06:** Issuer is NOT configurable — hardcoded for this project. No need for flexibility.

### Test Strategy
- **D-07:** RFC 6238 Appendix B test vectors (SHA-1 known answers) + edge case tests. Target ~15-20 tests:
  - RFC test vectors: known time → known OTP code
  - Time window validation: T-1, T, T+1 accepted; T-2, T+2 rejected
  - GenerateSecret: produces 20 bytes, valid base32 encoding
  - ValidateOTP: wrong code rejected, empty code rejected, malformed code rejected
  - GenerateProvisioningURI: correct format with URL-encoded email
  - Boundary: code at exact window transition (t=30, t=59, t=60)

### Claude's Discretion
- Internal HOTP helper function decomposition
- Error handling approach for GenerateSecret (crypto/rand failure)
- Whether ValidateOTP accepts `time.Time` or uses `time.Now()` internally (testing flexibility)
- Exact dynamic truncation implementation details

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Specification
- `workspace/03 - Security/TOTP RFC 6238.md` — Algorithm parameters, formula, provisioning URI format, verification rules

### Requirements
- `.planning/REQUIREMENTS.md` — CRYPTO-04, CRYPTO-05, CRYPTO-06, CRYPTO-07

### Architecture & Patterns
- `CLAUDE.md` — TOTP implementation constraints, no third-party libraries, zeroization rules
- `.planning/phases/04-shamir-secret-sharing/04-CONTEXT.md` — Established crypto package pattern (location, API style, test approach)

### Integration Points (Phase 7)
- `twofa/internal/services/twofaService/` — Will import totp package for 2FA setup/verify flow

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `twofa/internal/crypto/shamir/` — Established crypto package pattern (package-level functions, comprehensive tests)
- TwoFA Go module (`twofa/go.mod`) — module already exists, no setup needed
- `crypto/rand` already used in the project for random generation

### Established Patterns
- One concern per file (from Phase 4: `gf256.go`, `shamir.go`)
- Package-level functions for stateless crypto operations
- Standard `testing` package for pure math/crypto tests
- `init()` for table generation (GF256 pattern — may not apply to TOTP)

### Integration Points
- `twofa/internal/crypto/totp/` — new package, parallel to `shamir/`
- Phase 7 will import this package alongside shamir for the 2FA setup flow

</code_context>

<specifics>
## Specific Ideas

- HMAC-SHA1 from Go's `crypto/hmac` + `crypto/sha1` standard library (not third-party, but not from-scratch either — SHA-1 is not the academic focus, Shamir is)
- Time counter T = floor(unix_time / 30), encoded as 8-byte big-endian uint64
- Dynamic truncation: offset = hmac[19] & 0x0F, then 4 bytes masked with 0x7FFFFFFF, mod 10^6
- Base32 encoding without padding (standard for TOTP apps)

</specifics>

<deferred>
## Deferred Ideas

- **Secret zeroization** — Phase 7 handles clearing TOTP secret from memory after split
- **OTP single-use enforcement** — Phase 8 implements last-used counter tracking
- **Rate limiting** — Phase 8 handles 5-attempt-per-5-minutes via Redis

None — discussion stayed within phase scope

</deferred>

---

*Phase: 05-totp-implementation*
*Context gathered: 2026-04-12*
