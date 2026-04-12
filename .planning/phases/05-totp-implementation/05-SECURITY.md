---
phase: 05
slug: totp-implementation
status: verified
threats_open: 0
asvs_level: 1
created: 2026-04-12
---

# Phase 05 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| caller -> totp package | Caller provides secret bytes and time — library-level, no untrusted external input in this phase | TOTP secret ([]byte), unix timestamp (int64) |
| caller -> GenerateProvisioningURI | Email string from caller could contain special characters — must be URL-encoded | email (string), base32 secret (string) |

---

## Threat Register

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-05-01 | Information Disclosure | hotp string comparison | accept | Accepted: 6-digit code timing leak negligible. Additionally mitigated with `subtle.ConstantTimeCompare` at `totp.go:56-63`. Rate limiting in Phase 8. | closed |
| T-05-02 | Tampering | GenerateSecret | mitigate | `crypto/rand.Reader` used exclusively (`totp.go:23`). Zero `math/rand` imports in package — verified by grep. | closed |
| T-05-03 | Information Disclosure | Secret in logs | mitigate | No logging in totp package (pure library). Zero `slog` or `log.` usage — verified by grep. CLAUDE.md enforces never logging secrets. | closed |
| T-05-04 | Spoofing | Weak OTP validation | mitigate | Code length == 6 check (`totp.go:44`), digit-only validation (`totp.go:47-50`), exactly 3 time windows T-1/T/T+1 (`totp.go:56-65`). | closed |
| T-05-05 | Tampering | URI injection via email | mitigate | `url.PathEscape(email)` at `uri.go:14` prevents URI structure manipulation through `?`, `&`, `/` characters. | closed |
| T-05-06 | Information Disclosure | Secret in URI | accept | By design — otpauth:// standard requires secret in URI for QR code provisioning. URI is transient (shown to user, never persisted). Lifecycle managed in Phase 7. | closed |

*Status: open · closed*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-05-01 | T-05-01 | 6-digit OTP timing leak is negligible — 10^6 space with rate limiting (5 attempts/5min in Phase 8). Implementation additionally uses `subtle.ConstantTimeCompare` exceeding the accepted risk threshold. | gsd-secure-phase | 2026-04-12 |
| AR-05-02 | T-05-06 | Secret presence in otpauth:// URI is mandated by the provisioning standard (RFC 6238 + Google Authenticator Key URI Format). URI is generated transiently for QR display and never persisted to storage. | gsd-secure-phase | 2026-04-12 |

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-04-12 | 6 | 6 | 0 | gsd-secure-phase |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-04-12
