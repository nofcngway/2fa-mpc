---
phase: 7
slug: twofa-setup-flow
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-12
---

# Phase 7 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (standard library) |
| **Config file** | none — uses go test conventions |
| **Quick run command** | `cd twofa && go test ./internal/services/twofaService/... -count=1 -short` |
| **Full suite command** | `cd twofa && go test ./... -count=1 -v` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd twofa && go test ./internal/services/twofaService/... -count=1 -short`
- **After every plan wave:** Run `cd twofa && go test ./... -count=1 -v`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 07-01-01 | 01 | 1 | 2FA-01 | — | Setup2FA returns provisioning URI | unit | `cd twofa && go test ./internal/services/twofaService/... -run TestSetup -count=1` | ❌ W0 | ⬜ pending |
| 07-01-02 | 01 | 1 | 2FA-02 | — | Secret split into 3 shares, all stored | unit | `cd twofa && go test ./internal/services/twofaService/... -run TestShareDistribution -count=1` | ❌ W0 | ⬜ pending |
| 07-01-03 | 01 | 1 | SEC-04 | T-7-01 | Secret zeroized after distribution | unit | `cd twofa && go test ./internal/services/twofaService/... -run TestZeroization -count=1` | ❌ W0 | ⬜ pending |
| 07-02-01 | 02 | 1 | 2FA-08 | — | 10 backup codes generated, bcrypt-hashed | unit | `cd twofa && go test ./internal/services/twofaService/... -run TestBackupCodes -count=1` | ❌ W0 | ⬜ pending |
| 07-02-02 | 02 | 1 | 2FA-01 | — | Compensating delete on partial MPC failure | unit | `cd twofa && go test ./internal/services/twofaService/... -run TestPartialFailure -count=1` | ❌ W0 | ⬜ pending |
| 07-02-03 | 02 | 1 | 2FA-01 | — | Duplicate setup returns AlreadyExists | unit | `cd twofa && go test ./internal/services/twofaService/... -run TestDuplicateSetup -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `twofa/internal/services/twofaService/setup_test.go` — stubs for 2FA-01, 2FA-02, 2FA-08, SEC-04
- [ ] Mock interfaces for Storage, MPC clients

*Existing infrastructure covers crypto package tests (shamir, totp from Phases 4-5).*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Provisioning URI scannable by authenticator app | 2FA-01 | Requires physical device or authenticator emulator | Generate URI, scan with Google Authenticator or similar |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
