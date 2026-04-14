---
phase: 11
plan: 02
status: complete
subsystem: twofa
tags: [refactor, domain-model, error-consolidation]
dependency_graph:
  requires: []
  provides: [twofa-domain-package]
  affects: [twofa-service, twofa-api, twofa-storage, twofa-tests]
tech_stack:
  patterns: [domain-package, centralized-error-sentinels]
key_files:
  created:
    - twofa/internal/domain/models.go
    - twofa/internal/domain/errors.go
  modified:
    - twofa/internal/services/twofaService/twofa_service.go
    - twofa/internal/services/twofaService/verify.go
    - twofa/internal/services/twofaService/disable.go
    - twofa/internal/services/twofaService/status.go
    - twofa/internal/services/twofaService/setup.go
    - twofa/internal/services/twofaService/retrieve_shares.go
    - twofa/internal/services/twofaService/verify_backup_code.go
    - twofa/internal/services/twofaService/setup_test.go
    - twofa/internal/services/twofaService/verify_test.go
    - twofa/internal/services/twofaService/disable_test.go
    - twofa/internal/services/twofaService/verify_backup_code_test.go
    - twofa/internal/services/twofaService/status_test.go
    - twofa/internal/services/twofaService/mocks/storage_mock.go
    - twofa/internal/api/twofa_service_api/twofa_service_api.go
    - twofa/internal/api/twofa_service_api/disable.go
    - twofa/internal/api/twofa_service_api/verify.go
    - twofa/internal/api/twofa_service_api/setup.go
    - twofa/internal/storage/pgstorage/backup_code.go
    - twofa/internal/storage/pgstorage/twofa_record.go
    - twofa/internal/storage/redisstorage/noop.go
    - twofa/internal/storage/redisstorage/otp_counter.go
  deleted:
    - twofa/internal/models/models.go
decisions:
  - Grouped errors by category (state, verification, storage) following auth/internal/domain pattern
  - API handlers now import domain directly instead of referencing errors via twofaService package
  - Converted ErrInsufficientShares from fmt.Errorf to errors.New (static string, no formatting needed)
metrics:
  duration: 2577s
  completed: 2026-04-14
  tasks: 2
  files: 22
---

# Phase 11 Plan 02: Rename TwoFA internal/models to internal/domain + Consolidate Error Sentinels Summary

Renamed twofa/internal/models/ to twofa/internal/domain/ with split files (models.go for structs, errors.go for 8 consolidated error sentinels from 5 service files), updated 22 files across service, API, storage, and test layers.

## What was done

- Created `twofa/internal/domain/models.go` with 3 struct types (TwoFARecord, BackupCode, BackupCodeRow)
- Created `twofa/internal/domain/errors.go` with all 8 error sentinels grouped by category:
  - 2FA state errors: ErrAlreadyEnabled, ErrNotEnabled, ErrNotSetUp
  - Verification errors: ErrRateLimitExceeded, ErrOTPReused, ErrInsufficientShares, ErrInvalidBackupCode
  - Storage errors: ErrCounterNotFound
- Updated 7 service files to import `domain` and use `domain.Err*` instead of local sentinels
- Updated 3 API handler files to import `domain` directly instead of referencing errors through `twofaService` package
- Updated 5 test files to use `domain.Err*` and `domain.TwoFARecord` / `domain.BackupCodeRow`
- Updated 4 storage files (pgstorage + redisstorage) to import `domain`
- Regenerated minimock mocks (storage_mock.go now imports `domain`)
- Deleted old `twofa/internal/models/` directory (pb/models/ untouched)
- Removed 5 local `var Err*` declarations from service files
- Removed unused `"errors"` imports from setup.go and verify_backup_code.go
- Converted ErrInsufficientShares from `fmt.Errorf` to `errors.New` (static string)

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| 1 | b0731cb | Create twofa/internal/domain/ with models.go and errors.go |
| 2 | 140991f | Update all imports, remove local errors, regenerate mocks, delete old |

## Verification

- Build: PASS (`go build ./...`)
- Tests: PASS (`go test ./...` -- all twofaService tests pass including 24.8s suite)
- Import check: PASS (no remaining references to `twofa/internal/models` except pb/models)
- Error consolidation: 8/8 sentinels in domain/errors.go

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] API handler files also needed updating**
- **Found during:** Task 2
- **Issue:** Plan listed ~15 files but API handlers (disable.go, verify.go, setup.go in twofa_service_api/) referenced `twofaService.Err*` which no longer existed after error sentinels moved to domain package
- **Fix:** Updated 3 API handler files to import `domain` and use `domain.Err*` instead of `twofaService.Err*`
- **Files modified:** twofa/internal/api/twofa_service_api/{disable,verify,setup}.go

**2. [Rule 1 - Bug] Missing fmt import after removing fmt.Errorf sentinel**
- **Found during:** Task 2
- **Issue:** Removing the `fmt.Errorf` ErrInsufficientShares declaration from retrieve_shares.go also removed the `"fmt"` import, but `fmt.Errorf` was still used elsewhere in the file
- **Fix:** Re-added `"fmt"` to the import block
- **Files modified:** twofa/internal/services/twofaService/retrieve_shares.go
