---
phase: 2
slug: auth-registration
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-12
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (standard library) |
| **Config file** | none — Go test built-in |
| **Quick run command** | `cd auth && go test ./internal/services/authService/...` |
| **Full suite command** | `cd auth && go test ./...` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd auth && go test ./internal/services/authService/...`
- **After every plan wave:** Run `cd auth && go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 02-01-01 | 01 | 1 | AUTH-08 | — | Password validation rejects weak passwords | unit | `cd auth && go test ./internal/services/authService/ -run TestValidatePassword` | ❌ W0 | ⬜ pending |
| 02-01-02 | 01 | 1 | AUTH-08 | — | Sequential char detection (4+ consecutive) | unit | `cd auth && go test ./internal/services/authService/ -run TestValidatePassword` | ❌ W0 | ⬜ pending |
| 02-02-01 | 02 | 1 | AUTH-01 | — | User creation persists to PostgreSQL | unit | `cd auth && go test ./internal/services/authService/ -run TestRegister` | ❌ W0 | ⬜ pending |
| 02-02-02 | 02 | 1 | AUTH-02 | — | Duplicate email returns AlreadyExists | unit | `cd auth && go test ./internal/services/authService/ -run TestRegister` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `auth/internal/services/authService/password_validation_test.go` — table-driven tests for all password rules
- [ ] `auth/internal/services/authService/register_test.go` — registration tests with mocked storage

*Existing infrastructure covers test framework (go test built-in).*

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
