---
phase: 05-totp-implementation
plan: 02
subsystem: crypto
tags: [totp, otpauth, provisioning-uri, url-encoding, base32]

# Dependency graph
requires:
  - phase: 05-totp-implementation
    provides: TOTP core library (GenerateSecret, GenerateOTP, ValidateOTP)
provides:
  - "GenerateProvisioningURI function for authenticator app enrollment"
  - "otpauth:// URI generation with MPC-2FA issuer, URL-encoded email, base32 secret"
affects: [07-twofa-orchestration]

# Tech tracking
tech-stack:
  added: [net/url]
  patterns: [url-path-escape-for-labels, hardcoded-issuer-const]

key-files:
  created:
    - twofa/internal/crypto/totp/uri.go
  modified:
    - twofa/internal/crypto/totp/totp_test.go

key-decisions:
  - "Hardcoded issuer as package-level const, not configurable -- per D-06 decision"
  - "url.PathEscape for label (issuer:email), url.QueryEscape for query issuer param"

patterns-established:
  - "URI builder pattern: fmt.Sprintf with URL-escaped components, no url.URL struct needed for simple otpauth format"

requirements-completed: [CRYPTO-05, CRYPTO-07]

# Metrics
duration: 1min
completed: 2026-04-12
---

# Phase 5 Plan 02: Provisioning URI Summary

**GenerateProvisioningURI producing standard otpauth://totp/ URIs with MPC-2FA issuer, URL-encoded email, and all required TOTP parameters**

## Performance

- **Duration:** 1 min
- **Started:** 2026-04-12T06:43:07Z
- **Completed:** 2026-04-12T06:44:30Z
- **Tasks:** 1
- **Files modified:** 2

## Accomplishments
- Implemented GenerateProvisioningURI in uri.go with hardcoded MPC-2FA issuer constant
- Proper URL encoding: url.PathEscape for label components, url.QueryEscape for query params
- Added 4 URI tests (BasicFormat, SpecialCharsEmail, FullRoundtrip, EmptyEmail) -- all 17 tests pass

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement GenerateProvisioningURI and add tests** - `9f18110` (feat)

## Files Created/Modified
- `twofa/internal/crypto/totp/uri.go` - GenerateProvisioningURI with hardcoded MPC-2FA issuer, otpauth:// URI format
- `twofa/internal/crypto/totp/totp_test.go` - 4 new URI tests appended (BasicFormat, SpecialCharsEmail, FullRoundtrip, EmptyEmail)

## Decisions Made
- Hardcoded issuer as `const issuer = "MPC-2FA"` at package level per D-06 -- not configurable, matches project requirement
- Used `url.PathEscape` for both issuer and email in label segment, `url.QueryEscape` for issuer in query parameter

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Full TOTP crypto library complete: GenerateSecret, GenerateOTP, ValidateOTP, GenerateProvisioningURI
- Phase 7 (twofa-orchestration) can import `twofa/internal/crypto/totp` for 2FA setup flow
- No blockers or concerns

---
*Phase: 05-totp-implementation*
*Completed: 2026-04-12*
