---
phase: 8
slug: twofa-verification-management
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-12
---

# Phase 8 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (standard library) |
| **Config file** | none — go test built-in |
| **Quick run command** | `cd twofa && go test ./internal/services/twofaService/ -run "TestVerify\|TestDisable\|TestStatus\|TestRateLimit" -count=1` |
| **Full suite command** | `cd twofa && go test ./... -count=1` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run quick run command
- **After every plan wave:** Run full suite command
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 08-01-01 | 01 | 1 | 2FA-03 | — | Shares retrieved, combined, TOTP validated, secret zeroized | unit | `cd twofa && go test ./internal/services/twofaService/ -run TestVerify -count=1` | ❌ W0 | ⬜ pending |
| 08-01-02 | 01 | 1 | 2FA-04 | — | First successful verify enables 2FA (is_enabled=true) | unit | `cd twofa && go test ./internal/services/twofaService/ -run TestVerify_EnablesOn -count=1` | ❌ W0 | ⬜ pending |
| 08-01-03 | 01 | 1 | 2FA-09 | — | OTP reuse within same window rejected | unit | `cd twofa && go test ./internal/services/twofaService/ -run TestVerify_Reuse -count=1` | ❌ W0 | ⬜ pending |
| 08-01-04 | 01 | 1 | 2FA-05 | — | Rate limit enforced after 5 attempts in 5 min | unit | `cd twofa && go test ./internal/services/twofaService/ -run TestVerify_RateLimit -count=1` | ❌ W0 | ⬜ pending |
| 08-02-01 | 02 | 1 | 2FA-06 | — | Disable requires valid OTP, deletes shares + metadata | unit | `cd twofa && go test ./internal/services/twofaService/ -run TestDisable -count=1` | ❌ W0 | ⬜ pending |
| 08-02-02 | 02 | 1 | 2FA-07 | — | Status returns is_enabled + created_at | unit | `cd twofa && go test ./internal/services/twofaService/ -run TestStatus -count=1` | ❌ W0 | ⬜ pending |
| 08-02-03 | 02 | 1 | SEC-05 | — | Share data and encryption keys never logged | unit | `cd twofa && go test ./internal/services/twofaService/ -run TestVerify_NoSecretLog -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `twofa/internal/services/twofaService/verify_test.go` — stubs for 2FA-03, 2FA-04, 2FA-09
- [ ] `twofa/internal/services/twofaService/disable_test.go` — stubs for 2FA-06
- [ ] `twofa/internal/services/twofaService/status_test.go` — stubs for 2FA-07
- [ ] `twofa/internal/services/twofaService/rate_limit_test.go` — stubs for 2FA-05

*Existing test infrastructure from Phase 7 (setup_test.go with minimock patterns) covers framework needs.*

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
