# Phase 5: TOTP Implementation - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-12
**Phase:** 05-totp-implementation
**Areas discussed:** Package location, API & file structure, Test strategy, Provisioning URI

---

## Package Location

| Option | Description | Selected |
|--------|-------------|----------|
| twofa/internal/crypto/totp/ | Follows Phase 4 pattern — crypto packages grouped under internal/crypto/. Consistent with shamir/ | ✓ |
| twofa/internal/services/twofaService/totp/ | As documented in workspace/03 - Security/TOTP RFC 6238.md. Nested inside service dir | |
| twofa/pkg/totp/ | Public package — importable by other modules directly. More Go-idiomatic for libraries | |

**User's choice:** twofa/internal/crypto/totp/ (Recommended)
**Notes:** Consistency with Phase 4 Shamir pattern was the deciding factor.

---

## API & File Structure

### File organization

| Option | Description | Selected |
|--------|-------------|----------|
| Single totp.go + totp_test.go | TOTP is compact (~100-150 lines). One file keeps it simple | |
| Split: hotp.go + totp.go + totp_test.go | Separate HOTP core from TOTP time wrapper. Clearer layering for  | |
| Split: totp.go + uri.go + totp_test.go | Separate provisioning URI generation. Main logic vs presentation concern | ✓ |

**User's choice:** Split: totp.go + uri.go + totp_test.go
**Notes:** URI generation is a presentation concern separate from crypto logic.

### API surface

| Option | Description | Selected |
|--------|-------------|----------|
| 4 functions | GenerateSecret, GenerateOTP, ValidateOTP, GenerateProvisioningURI | ✓ |
| 4 functions + HOTP export | Same plus exported GenerateHOTP for  layering | |
| You decide | Claude chooses minimal API | |

**User's choice:** 4 functions (Recommended)
**Notes:** HOTP stays as internal helper, not exported.

---

## Test Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| RFC vectors + edge cases | RFC 6238 Appendix B test vectors + boundary tests. ~15-20 tests | ✓ |
| Maximum coverage | RFC vectors + edge cases + property tests. ~25+ tests | |
| RFC vectors only | Only official test vectors. ~5-8 tests | |

**User's choice:** RFC vectors + edge cases (Recommended)
**Notes:** Sufficient for  without being excessive.

---

## Provisioning URI

| Option | Description | Selected |
|--------|-------------|----------|
| MPC-2FA:{email} | Fixed issuer, standard Google Authenticator format per workspace spec | ✓ |
| Configurable issuer | Pass issuer as parameter. More flexible but unnecessary complexity | |
| You decide | Claude picks based on RFC and workspace docs | |

**User's choice:** MPC-2FA:{email} (Recommended)
**Notes:** No configurability needed — single-purpose academic project.

---

## Claude's Discretion

- Internal HOTP helper function decomposition
- Error handling approach for GenerateSecret
- Whether ValidateOTP uses time injection for testability
- Dynamic truncation implementation details

## Deferred Ideas

- Secret zeroization (Phase 7)
- OTP single-use enforcement (Phase 8)
- Rate limiting (Phase 8)
