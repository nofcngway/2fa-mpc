---
phase: 6
slug: mpc-node-service
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-12
---

# Phase 6 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test + gotest.tools/v3 + minimock/v3 |
| **Config file** | none — existing test setup in mpc/ |
| **Quick run command** | `cd mpc && go test ./internal/...` |
| **Full suite command** | `cd mpc && go test -v -count=1 ./...` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd mpc && go test ./internal/...`
- **After every plan wave:** Run `cd mpc && go test -v -count=1 ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 06-01-01 | 01 | 1 | MPC-01 | T-06-01 | AES-256-GCM encryption with unique nonce per share | unit | `cd mpc && go test ./internal/services/mpcService/ -run TestEncrypt` | ❌ W0 | ⬜ pending |
| 06-01-02 | 01 | 1 | MPC-02 | T-06-02 | Nonce generated via crypto/rand, 12 bytes | unit | `cd mpc && go test ./internal/services/mpcService/ -run TestNonce` | ❌ W0 | ⬜ pending |
| 06-01-03 | 01 | 1 | MPC-03 | — | StoreShare persists encrypted_data + nonce in PostgreSQL | integration | `cd mpc && go test ./internal/storage/pgstorage/ -run TestStoreShare` | ❌ W0 | ⬜ pending |
| 06-01-04 | 01 | 1 | MPC-03 | — | Duplicate (user_id, share_index) rejected by unique constraint | integration | `cd mpc && go test ./internal/storage/pgstorage/ -run TestDuplicate` | ❌ W0 | ⬜ pending |
| 06-02-01 | 02 | 1 | MPC-04 | T-06-03 | RetrieveShare decrypts and returns original data | unit | `cd mpc && go test ./internal/services/mpcService/ -run TestRetrieve` | ❌ W0 | ⬜ pending |
| 06-02-02 | 02 | 1 | MPC-05 | — | DeleteShare removes all shares for user | integration | `cd mpc && go test ./internal/storage/pgstorage/ -run TestDeleteShare` | ❌ W0 | ⬜ pending |
| 06-02-03 | 02 | 1 | MPC-06 | T-06-04 | Requests without valid shared secret rejected by interceptor | unit | `cd mpc && go test ./internal/middleware/ -run TestAuthInterceptor` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `mpc/internal/services/mpcService/*_test.go` — unit tests for encryption/decryption
- [ ] `mpc/internal/storage/pgstorage/*_test.go` — storage layer tests
- [ ] `mpc/internal/middleware/*_test.go` — auth interceptor tests
- [ ] Add `gotest.tools/v3`, `minimock/v3`, `google/uuid` to `mpc/go.mod`

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Encryption key from config.yaml is 32 bytes | MPC-01 | Config validation at bootstrap | Verify bootstrap panics with invalid key length |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
