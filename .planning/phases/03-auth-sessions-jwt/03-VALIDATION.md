---
phase: 03
slug: auth-sessions-jwt
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-12
---

# Phase 03 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (standard library) + minimock + gotest.tools/v3/assert |
| **Config file** | none — existing test infrastructure from Phase 2 |
| **Quick run command** | `cd auth && go test ./internal/services/authService/... -count=1` |
| **Full suite command** | `cd auth && go test ./... -count=1 -v` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd auth && go test ./internal/services/authService/... -count=1`
- **After every plan wave:** Run `cd auth && go test ./... -count=1 -v`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 03-01-01 | 01 | 1 | AUTH-03, SEC-01 | T-03-01 | JWT RS256 sign/verify with algorithm confusion prevention | unit | `cd auth && go test ./internal/services/authService/... -run TestJWT -count=1` | ❌ W0 | ⬜ pending |
| 03-01-02 | 01 | 1 | AUTH-03 | — | Redis session storage CRUD operations | unit | `cd auth && go test ./internal/services/authService/... -run TestSession -count=1` | ❌ W0 | ⬜ pending |
| 03-02-01 | 02 | 2 | AUTH-03, SEC-03 | T-03-02 | Login with credential validation, no password in response | unit | `cd auth && go test ./internal/services/authService/... -run TestLogin -count=1` | ❌ W0 | ⬜ pending |
| 03-02-02 | 02 | 2 | AUTH-04, AUTH-05 | T-03-03 | Refresh token rotation with theft detection | unit | `cd auth && go test ./internal/services/authService/... -run TestRefresh -count=1` | ❌ W0 | ⬜ pending |
| 03-02-03 | 02 | 2 | AUTH-06 | — | Logout single and logout-all | unit | `cd auth && go test ./internal/services/authService/... -run TestLogout -count=1` | ❌ W0 | ⬜ pending |
| 03-02-04 | 02 | 2 | AUTH-07, SEC-01 | T-03-01 | Token validation with algorithm restriction | unit | `cd auth && go test ./internal/services/authService/... -run TestValidate -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

*Existing infrastructure covers all phase requirements. Phase 2 already established go test + minimock + gotest.tools/v3/assert.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| RSA key loading from file | AUTH-03 | Requires actual key files on disk | Run `make generate-keys` then `go test` with key path in test config |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
