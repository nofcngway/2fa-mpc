---
phase: 5
slug: totp-implementation
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-12
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` (standard library) |
| **Config file** | None — `go test` works out of the box |
| **Quick run command** | `cd twofa && go test ./internal/crypto/totp/ -v -count=1` |
| **Full suite command** | `cd twofa && go test ./internal/crypto/... -v -count=1` |
| **Estimated runtime** | ~2 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd twofa && go test ./internal/crypto/totp/ -v -count=1`
- **After every plan wave:** Run `cd twofa && go test ./internal/crypto/... -v -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 05-01-01 | 01 | 1 | CRYPTO-04 | T-05-01 | crypto/rand for secret generation | unit | `cd twofa && go test ./internal/crypto/totp/ -run TestGenerateSecret -v` | ❌ W0 | ⬜ pending |
| 05-01-02 | 01 | 1 | CRYPTO-04 | T-05-02 | HMAC-SHA1 dynamic truncation per RFC | unit | `cd twofa && go test ./internal/crypto/totp/ -run TestGenerateOTP_RFC6238 -v` | ❌ W0 | ⬜ pending |
| 05-01-03 | 01 | 1 | CRYPTO-06 | — | ±1 window validation | unit | `cd twofa && go test ./internal/crypto/totp/ -run TestValidateOTP_TimeWindow -v` | ❌ W0 | ⬜ pending |
| 05-01-04 | 01 | 1 | CRYPTO-05 | — | Valid otpauth:// URI format | unit | `cd twofa && go test ./internal/crypto/totp/ -run TestGenerateProvisioningURI -v` | ❌ W0 | ⬜ pending |
| 05-01-05 | 01 | 1 | CRYPTO-07 | — | Edge cases: wrong/empty/malformed codes | unit | `cd twofa && go test ./internal/crypto/totp/ -run TestValidateOTP -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `twofa/internal/crypto/totp/totp.go` — core TOTP implementation
- [ ] `twofa/internal/crypto/totp/uri.go` — provisioning URI generation
- [ ] `twofa/internal/crypto/totp/totp_test.go` — all tests (RFC vectors + edge cases)

*Existing infrastructure covers framework needs — Go testing is built-in.*

---

## Manual-Only Verifications

All phase behaviors have automated verification.

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
