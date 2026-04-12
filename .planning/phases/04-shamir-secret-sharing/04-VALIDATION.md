---
phase: 4
slug: shamir-secret-sharing
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-12
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib) |
| **Config file** | none — standard Go testing |
| **Quick run command** | `cd twofa && go test ./internal/crypto/shamir/ -count=1` |
| **Full suite command** | `cd twofa && go test ./internal/crypto/shamir/ -v -count=1` |
| **Estimated runtime** | ~2 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd twofa && go test ./internal/crypto/shamir/ -count=1`
- **After every plan wave:** Run `cd twofa && go test ./internal/crypto/shamir/ -v -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 2 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 04-01-01 | 01 | 1 | CRYPTO-01 | — | GF(256) field axioms hold for all 256 elements | unit | `cd twofa && go test ./internal/crypto/shamir/ -run TestGF256 -count=1` | ❌ W0 | ⬜ pending |
| 04-01-02 | 01 | 1 | CRYPTO-02 | — | Split produces n distinct shares | unit | `cd twofa && go test ./internal/crypto/shamir/ -run TestSplit -count=1` | ❌ W0 | ⬜ pending |
| 04-02-01 | 02 | 1 | CRYPTO-02 | — | Combine 2-of-3 recovers secret | unit | `cd twofa && go test ./internal/crypto/shamir/ -run TestCombine -count=1` | ❌ W0 | ⬜ pending |
| 04-02-02 | 02 | 1 | CRYPTO-03 | — | Combine 1-of-3 does NOT recover | unit | `cd twofa && go test ./internal/crypto/shamir/ -run TestCombineInsufficient -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `twofa/internal/crypto/shamir/` — directory creation
- [ ] `twofa/internal/crypto/shamir/gf256_test.go` — GF(256) property test stubs
- [ ] `twofa/internal/crypto/shamir/shamir_test.go` — Split/Combine test stubs

*Existing Go test infrastructure covers framework needs.*

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 2s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
