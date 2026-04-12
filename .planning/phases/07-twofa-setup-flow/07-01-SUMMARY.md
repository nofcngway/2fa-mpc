---
phase: 07-twofa-setup-flow
plan: 01
subsystem: twofa
tags: [proto, interfaces, storage, mpc-client, bootstrap, zeroize]
dependency_graph:
  requires: [mpc-proto-contract]
  provides: [twofa-storage-interface, twofa-mpc-client-interface, twofa-pgstorage-crud, twofa-mpc-wiring, twofa-zeroize-utility]
  affects: [twofa-service, twofa-bootstrap, twofa-config]
tech_stack:
  added: [minimock]
  patterns: [cross-module-proto-generation, auth-metadata-interceptor]
key_files:
  created:
    - twofa/internal/crypto/zeroize.go
    - twofa/internal/crypto/zeroize_test.go
    - twofa/internal/storage/pgstorage/twofa_record.go
    - twofa/internal/storage/pgstorage/backup_code.go
    - twofa/internal/services/twofaService/setup.go
    - twofa/internal/services/twofaService/mocks/storage_mock.go
    - twofa/internal/services/twofaService/mocks/mpc_client_mock.go
    - twofa/api/mpc_api/mpc_service.proto
    - twofa/internal/pb/mpc_api/mpc_service.pb.go
    - twofa/internal/pb/mpc_api/mpc_service_grpc.pb.go
  modified:
    - twofa/api/twofa_api/twofa_service.proto
    - twofa/internal/pb/twofa_api/twofa_service.pb.go
    - twofa/internal/pb/twofa_api/twofa_service_grpc.pb.go
    - twofa/internal/services/twofaService/twofa_service.go
    - twofa/internal/api/twofa_service_api/twofa_service_api.go
    - twofa/internal/bootstrap/bootstrap.go
    - twofa/config/config.go
    - twofa/cmd/app/main.go
    - twofa/go.mod
    - twofa/go.sum
decisions:
  - Generated MPC proto locally in twofa module instead of importing from mpc/internal (Go internal package restriction)
  - Added Setup method stub for interface satisfaction (full implementation in Plan 02)
metrics:
  duration: 394s
  completed: "2026-04-12T09:17:11Z"
  tasks_completed: 2
  tasks_total: 2
  files_created: 10
  files_modified: 10
---

# Phase 07 Plan 01: TwoFA Setup Foundations Summary

Proto update with email field, zeroize utility, Storage/MPCClient interfaces, PGStorage CRUD, MPC client wiring with auth interceptor, and minimock generation for testability.

## Task Results

| Task | Name | Commit | Status |
|------|------|--------|--------|
| 1 | Proto update, zeroize utility, and dependency setup | 4d12808 | Done |
| 2 | Storage methods, interfaces, MPC client wiring, and bootstrap update | dcdce3c | Done |

## What Was Built

### Task 1: Proto + Zeroize + Dependencies
- Added `string email = 2` to `Setup2FARequest` in twofa_service.proto
- Regenerated protobuf Go code with Email field
- Created `Zeroize([]byte)` utility for clearing secrets from memory
- All 4 zeroize test subtests pass (zeroes bytes, empty slice, nil slice, preserves length)

### Task 2: Interfaces + Storage + MPC Wiring + Bootstrap
- **Storage interface**: `CreateTwoFARecord`, `GetTwoFARecord`, `StoreBatchBackupCodes`
- **MPCClient interface**: mirrors `mpc_api.MPCNodeServiceClient` with `StoreShare`, `RetrieveShare`, `DeleteShare`
- **PGStorage.CreateTwoFARecord**: INSERT with `is_enabled=FALSE`, parameterized query
- **PGStorage.GetTwoFARecord**: SELECT with `nil, nil` return on `pgx.ErrNoRows`
- **PGStorage.StoreBatchBackupCodes**: transactional INSERT with `uuid.New()` per code
- **NewMPCClients bootstrap**: creates gRPC connections with `insecure.NewCredentials()` + auth metadata interceptor
- **authMetadataInterceptor**: attaches shared secret in "authorization" metadata on every outgoing call
- **Config.MPCTimeout**: `time.Duration` field with `GetMPCTimeout()` returning 5s default
- **TwoFAService constructor**: updated to accept `mpcClients`, `sharedSecret`, `mpcTimeout`
- **Mocks generated**: `storage_mock.go`, `mpc_client_mock.go` via minimock

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Go internal package restriction prevents cross-module import**
- **Found during:** Task 2
- **Issue:** Plan specified importing `github.com/vbncursed/vkr/mpc/internal/pb/mpc_api` from the twofa module, but Go's `internal` package restriction prevents cross-module access to internal packages.
- **Fix:** Generated MPC proto code locally within the twofa module (`twofa/api/mpc_api/mpc_service.proto` with `go_package` pointing to `twofa/internal/pb/mpc_api`). This produces identical Go types within twofa's own module scope.
- **Files created:** `twofa/api/mpc_api/mpc_service.proto`, `twofa/internal/pb/mpc_api/*.pb.go`
- **Commit:** dcdce3c

**2. [Rule 3 - Blocking] main.go not updated for new bootstrap signature**
- **Found during:** Task 2
- **Issue:** `bootstrap.NewTwoFAService` signature changed to require `mpcClients` and `cfg`, but `cmd/app/main.go` still used old 2-arg call.
- **Fix:** Updated main.go to call `NewMPCClients` and pass results to `NewTwoFAService`.
- **Files modified:** `twofa/cmd/app/main.go`
- **Commit:** dcdce3c

**3. [Rule 3 - Blocking] TwoFAService missing Setup method for Service interface**
- **Found during:** Task 2
- **Issue:** API layer's `Service` interface requires `Setup` method but `TwoFAService` doesn't implement it yet (Plan 02 scope).
- **Fix:** Added stub `setup.go` returning `errors.New("not implemented")` to satisfy the interface and allow compilation.
- **Files created:** `twofa/internal/services/twofaService/setup.go`
- **Commit:** dcdce3c

## Known Stubs

| File | Line | Reason |
|------|------|--------|
| `twofa/internal/services/twofaService/setup.go` | 12 | Setup method returns "not implemented" - full orchestration logic in Plan 02 |
| `twofa/internal/services/twofaService/twofa_service.go` | 26 | SessionStorage interface has no methods - added in Phase 8 (rate limiting) |

## Threat Flags

None - all security surfaces match the plan's threat model. Auth metadata interceptor (T-07-01), parameterized queries (T-07-03), and zeroize utility (T-07-02) are implemented as mitigations.

## Self-Check: PASSED

All 7 created files verified on disk. Both commit hashes (4d12808, dcdce3c) found in git log.
