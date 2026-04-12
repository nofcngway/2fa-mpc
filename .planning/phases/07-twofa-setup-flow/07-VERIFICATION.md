---
phase: 07-twofa-setup-flow
verified: 2026-04-12T11:45:00Z
status: gaps_found
score: 4/4
overrides_applied: 0
gaps:
  - truth: "17 tests exist covering Setup2FA scenarios — plan behavior spec named TestSetup_SecretZeroized as a required test but it is absent"
    status: partial
    reason: "TestSetup_SecretZeroized was explicitly listed in plan 02 behavior spec and threat model (T-07-01 mitigation evidence). Only TestSetup_SharesZeroized exists, and that test does not assert byte-level zeroing — it only verifies the service completes. The zeroize code path exists (defer crypto.Zeroize(raw)) and Zeroize is independently tested, but the named test from the acceptance criteria is missing."
    artifacts:
      - path: "twofa/internal/services/twofaService/setup_test.go"
        issue: "TestSetup_SecretZeroized absent; TestSetup_SharesZeroized exists but only asserts err==nil, not that raw bytes are zeroed"
    missing:
      - "Add TestSetup_SecretZeroized that captures raw bytes before Setup, runs Setup, and asserts all bytes are zero after return"
  - truth: "session_storage_mock.go generated — plan 01 acceptance criteria listed twofa/internal/services/twofaService/mocks/session_storage_mock.go"
    status: failed
    reason: "Plan 01 acceptance criteria explicitly listed mocks/session_storage_mock.go. It is absent from the mocks directory. The go:generate directive for it exists in twofa_service.go but was not run (or minimock skipped empty interfaces). Tests pass with nil for sessionStorage."
    artifacts:
      - path: "twofa/internal/services/twofaService/mocks/session_storage_mock.go"
        issue: "File does not exist"
    missing:
      - "Run 'cd twofa && go generate ./internal/services/twofaService/' to generate session_storage_mock.go"
---

# Phase 7: TwoFA Setup Flow Verification Report

**Phase Goal:** Users can enable 2FA with their TOTP secret securely split and distributed across MPC nodes
**Verified:** 2026-04-12T11:45:00Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Roadmap Observable Truths

All four roadmap success criteria are verified. Two plan acceptance-criteria items are gaps.

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | User calls Setup2FA and receives a provisioning URI and 10 backup codes | VERIFIED | TestSetup_Success passes: returns non-empty URI containing `otpauth://totp/` + email, exactly 10 codes. All 17 tests pass (`go test ./internal/services/twofaService/ -count=1`, 25.6s). |
| 2 | TOTP secret split into 3 shares, all 3 stored across MPC nodes; any unreachable node causes setup to fail completely | VERIFIED | `distributeShares` uses `errgroup.WithContext` for parallel distribution to 3 `mpcClients`. On ANY error `g.Wait()` returns the first error and `deleteSharesFromAllNodes` is called. TestSetup_PartialMPCFailure_Node2Fails, _Node0Fails, and TestSetup_AllMPCNodesFail all pass — each verifies DeleteShare called on all 3 nodes after failure. |
| 3 | TOTP secret zeroized from memory after share distribution — never persisted whole | VERIFIED (code) | `setup.go:48` has `defer crypto.Zeroize(raw)` immediately after `totp.GenerateSecret()`. Shares' Data bytes zeroed via deferred loop (`setup.go:56-60`). `Zeroize` utility tested by 4 subtests (TestZeroize passes). Secret bytes never written to PostgreSQL or Kafka. Note: TestSetup_SecretZeroized (plan-named test) is absent — see gaps. |
| 4 | Backup codes are bcrypt-hashed before storage in PostgreSQL | VERIFIED | `generateBackupCodes()` calls `bcrypt.GenerateFromPassword([]byte(code), COST_BCRYPT)` where `COST_BCRYPT=12`. `StoreBatchBackupCodes` receives hashed strings, not plaintext. TestSetup_BackupCodeHashing verifies `bcrypt.CompareHashAndPassword` succeeds for each of 10 codes. |

**Score:** 4/4 roadmap truths verified

### Plan Must-Haves — Plan 01 (foundations)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Proto Setup2FARequest contains email field (field number 2) | VERIFIED | `twofa/api/twofa_api/twofa_service.proto:16` has `string email = 2;`; pb.go:28 has `Email string` field. |
| 2 | Storage interface has CreateTwoFARecord, GetTwoFARecord, StoreBatchBackupCodes | VERIFIED | `twofa_service.go:17-22` — all 3 methods in Storage interface. |
| 3 | MPCClient interface mirrors mpc_api.MPCNodeServiceClient (StoreShare, RetrieveShare, DeleteShare) | VERIFIED | `twofa_service.go:30-35` — all 3 methods with matching signatures using local `mpc_api` protobuf types. |
| 4 | Bootstrap creates 3 MPC gRPC client connections from config.yaml mpc_nodes array | VERIFIED | `bootstrap.go:42-61` — `NewMPCClients` loops `cfg.MPCNodes`, calls `grpc.NewClient` with `insecure.NewCredentials()` + `authMetadataInterceptor`. |
| 5 | Zeroize utility sets all bytes in slice to zero | VERIFIED | `zeroize.go:4-9` — simple range loop zeroing. TestZeroize 4 subtests all PASS. |
| 6 | TwoFAService struct holds mpcClients, sharedSecret, mpcTimeout fields | VERIFIED | `twofa_service.go:37-44` — struct has `mpcClients []MPCClient`, `sharedSecret string`, `mpcTimeout time.Duration`. |

### Plan Must-Haves — Plan 02 (orchestration)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | User calls Setup2FA and receives provisioning URI and 10 backup codes | VERIFIED | See roadmap truth #1. |
| 2 | TOTP secret split into 3 shares, all 3 stored across MPC nodes in parallel | VERIFIED | `errgroup.WithContext` + goroutine per mpcClient, parallel `StoreShare` calls. |
| 3 | Any MPC node unreachable causes setup to fail with compensating delete on all nodes | VERIFIED | `distributeShares` calls `deleteSharesFromAllNodes` on errgroup failure. Uses `context.Background()` (not cancelled context) — verified by TestSetup_CompensatingDeleteUsesFreshContext. |
| 4 | TOTP secret zeroized from memory after share distribution | VERIFIED | `defer crypto.Zeroize(raw)` at `setup.go:48`. |
| 5 | Share Data bytes zeroized from memory after distribution | VERIFIED | Deferred loop at `setup.go:56-60` calls `crypto.Zeroize(shares[i].Data)` for each share. |
| 6 | Backup codes bcrypt-hashed before storage, plaintext returned only in response | VERIFIED | `generateBackupCodes` hashes via bcrypt cost=12; `StoreBatchBackupCodes` receives hashes; plaintext returned to caller, never persisted. |
| 7 | Duplicate setup with is_enabled=true returns AlreadyExists error | VERIFIED | `setup.go:39` checks `existing != nil && existing.IsEnabled` returning `ErrAlreadyEnabled`. TestSetup_DuplicateEnabled passes. gRPC handler maps to `codes.AlreadyExists`. |
| 8 | Backup codes match format xxxx-xxxx (8 digits with hyphen) | VERIFIED | `backup_codes.go:29` uses `fmt.Sprintf("%04d-%04d", ...)`. TestSetup_BackupCodeFormat regex `^\d{4}-\d{4}$` passes for all 10. |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `twofa/internal/crypto/zeroize.go` | Zeroize byte slice utility | VERIFIED | 9 lines, `func Zeroize(b []byte)` with range-loop zero-fill |
| `twofa/internal/crypto/zeroize_test.go` | Zeroize unit test | VERIFIED | TestZeroize with 4 subtests, all pass |
| `twofa/internal/storage/pgstorage/twofa_record.go` | CreateTwoFARecord and GetTwoFARecord | VERIFIED | Both methods implemented with parameterized queries, `nil, nil` on ErrNoRows |
| `twofa/internal/storage/pgstorage/backup_code.go` | StoreBatchBackupCodes | VERIFIED | Transactional INSERT with uuid.New() per code |
| `twofa/internal/services/twofaService/twofa_service.go` | Storage, MPCClient interfaces and updated TwoFAService constructor | VERIFIED | All 3 interfaces defined, constructor takes mpcClients/sharedSecret/mpcTimeout |
| `twofa/internal/services/twofaService/setup.go` | Setup2FA orchestration (>60 lines) | VERIFIED | 140 lines, full orchestration including errgroup, zeroize, compensating delete |
| `twofa/internal/services/twofaService/backup_codes.go` | Backup code generation helper | VERIFIED | generateBackupCodes/generateBackupCode with crypto/rand and bcrypt cost=12 |
| `twofa/internal/services/twofaService/setup_test.go` | Comprehensive tests (>200 lines) | VERIFIED | 497 lines, 17 test functions, all pass |
| `twofa/internal/api/twofa_service_api/setup.go` | Setup2FA gRPC handler | VERIFIED | Input validation, ErrAlreadyEnabled -> AlreadyExists, generic internal error for other errors |
| `twofa/internal/bootstrap/bootstrap.go` | NewMPCClients, authMetadataInterceptor | VERIFIED | Both functions present and wired |
| `twofa/internal/services/twofaService/mocks/storage_mock.go` | Minimock-generated Storage mock | VERIFIED | Exists, used by setup_test.go |
| `twofa/internal/services/twofaService/mocks/mpc_client_mock.go` | Minimock-generated MPCClient mock | VERIFIED | Exists, used by setup_test.go |
| `twofa/internal/services/twofaService/mocks/session_storage_mock.go` | Minimock-generated SessionStorage mock | MISSING | Directory only contains storage_mock.go and mpc_client_mock.go — see gaps |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `twofa/internal/bootstrap/bootstrap.go` | `mpc/internal/pb/mpc_api` (local copy at `twofa/internal/pb/mpc_api`) | `grpc.NewClient + NewMPCNodeServiceClient` | VERIFIED | `bootstrap.go:57` calls `mpc_api.NewMPCNodeServiceClient(conn)` |
| `twofa/internal/services/twofaService/twofa_service.go` | `twofa/internal/storage/pgstorage` | Storage interface implementation | VERIFIED | PGStorage implements CreateTwoFARecord, GetTwoFARecord, StoreBatchBackupCodes — all methods present |
| `twofa/internal/services/twofaService/setup.go` | `twofa/internal/crypto/shamir` | `shamir.Split(raw, 3, 2)` | VERIFIED | `setup.go:51` calls `shamir.Split(raw, 3, 2)` |
| `twofa/internal/services/twofaService/setup.go` | `twofa/internal/crypto/totp` | `totp.GenerateSecret()` and `totp.GenerateProvisioningURI()` | VERIFIED | `setup.go:44` and `setup.go:85` |
| `twofa/internal/services/twofaService/setup.go` | `twofa/internal/crypto` | `crypto.Zeroize(raw)` and `crypto.Zeroize(shares[i].Data)` | VERIFIED | `setup.go:48` and `setup.go:58` |
| `twofa/internal/services/twofaService/setup.go` | MPCClient interface | errgroup parallel StoreShare calls | VERIFIED | `setup.go:105` — `s.mpcClients[i].StoreShare(callCtx, ...)` |
| `twofa/internal/api/twofa_service_api/setup.go` | `twofa/internal/services/twofaService` | `api.service.Setup(ctx, req.UserId, req.Email)` | VERIFIED | `setup.go:20` |

### Data-Flow Trace (Level 4)

The Setup2FA flow is orchestration logic, not a data-rendering component. Data flows are traced at the logic level:

| Stage | Data Variable | Source | Produces Real Data | Status |
|-------|---------------|--------|--------------------|--------|
| TOTP generation | `raw, base32Secret` | `totp.GenerateSecret()` — crypto/rand 20 bytes | Yes | FLOWING |
| Shamir split | `shares []shamir.Share` | `shamir.Split(raw, 3, 2)` | Yes | FLOWING |
| MPC distribution | StoreShareRequest | `s.mpcClients[i].StoreShare(...)` | Yes | FLOWING |
| Backup codes | `plaintextCodes, hashedCodes` | `generateBackupCodes()` — crypto/rand + bcrypt | Yes | FLOWING |
| Provisioning URI | `uri` | `totp.GenerateProvisioningURI(base32Secret, email)` | Yes | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Module compiles | `cd twofa && go build ./...` | Exit 0, no output | PASS |
| Zeroize tests pass | `go test ./internal/crypto/ -v -count=1` | TestZeroize: 4/4 subtests PASS | PASS |
| Setup tests pass (17 tests) | `go test ./internal/services/twofaService/ -count=1` | ok 25.6s | PASS |
| Handler has no Unimplemented | `grep 'Unimplemented' setup.go` | No matches | PASS |
| Secret logging absent | `grep 'slog.*raw\|slog.*share\|slog.*secret'` in setup.go | No matches | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|---------|
| 2FA-01 | 07-01, 07-02 | User can setup 2FA — TOTP secret generated, split via Shamir (2-of-3), shares sent to 3 MPC nodes, secret zeroized, provisioning URI returned | SATISFIED | Full Setup orchestration in setup.go; TestSetup_Success passes |
| 2FA-02 | 07-01, 07-02 | Setup fails if any MPC node is unreachable (all 3 shares MUST be stored) | SATISFIED | errgroup parallel distribution; partial failure tests pass |
| 2FA-08 | 07-01, 07-02 | 10 backup codes generated on setup, each bcrypt-hashed, stored in PostgreSQL | SATISFIED | generateBackupCodes returns 10 codes; StoreBatchBackupCodes stores hashes; TestSetup_BackupCodeHashing passes |
| SEC-04 | 07-01, 07-02 | TOTP secret never persisted — only transient in memory, zeroized after use | SATISFIED | defer crypto.Zeroize(raw) in setup.go; shares zeroed after distribution; no DB writes of raw secret |

No orphaned requirements: REQUIREMENTS.md traceability table maps 2FA-01, 2FA-02, 2FA-08, SEC-04 all to Phase 7 — all four are covered.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `twofa/internal/services/twofaService/setup_test.go` | 479-483 | TestSetup_SharesZeroized only asserts `err == nil` — no actual byte-level zero assertion | Warning | Zeroization of share data is not programmatically verified by tests; relies on code review |
| `mocks/session_storage_mock.go` | — | Missing file (go:generate exists, mock not generated) | Warning | SessionStorage has no methods so tests pass with nil; no current functional impact |

No TODO/FIXME/placeholder patterns found in production code files. No hardcoded empty returns. No stub implementations in the Setup2FA path (stub from Plan 01 was correctly replaced in Plan 02).

### Human Verification Required

None — all key behaviors are verifiable programmatically. The Setup2FA flow is pure Go logic with no visual or external-service dependencies at this phase.

### Gaps Summary

Two plan acceptance-criteria failures exist. The phase goal (users can enable 2FA with secret distributed across MPC nodes) is fully achieved — all 4 roadmap success criteria pass. The gaps are minor plan-compliance items:

**Gap 1 — Missing TestSetup_SecretZeroized test:** Plan 02 behavior spec explicitly named this test. The code correctly implements `defer crypto.Zeroize(raw)` and the Zeroize utility is verified by its own test suite. The missing test is a test-coverage gap, not a logic gap. Fix: add a test that captures the `raw` pointer before Setup and asserts all bytes are zero after Setup returns (requires a seam in `totp.GenerateSecret` or a different testing approach such as monkey-patching or an injectable generator).

**Gap 2 — Missing session_storage_mock.go:** Plan 01 acceptance criteria listed this file. The `SessionStorage` interface has no methods (deliberately empty pending Phase 8). Running `cd twofa && go generate ./internal/services/twofaService/` should generate it. Fix: run the generate command. No tests are currently broken without it.

Both gaps are immediately fixable and do not affect the correctness of the implemented Setup2FA flow.

---

_Verified: 2026-04-12T11:45:00Z_
_Verifier: Claude (gsd-verifier)_
