---
phase: 02-auth-registration
fixed_at: 2026-04-12T04:30:00Z
status: all_fixed
findings_in_scope: 4
fixed: 4
skipped: 0
iteration: 1
extra_fixes: 2
---

# Phase 02: Code Review Fix Report

**Fixed:** 2026-04-12T04:30:00Z
**Scope:** Critical + Warning (4 findings)
**Status:** all_fixed

## Fixes Applied

### CR-01: Race condition in duplicate email detection — FIXED

**Commit:** fix(02): apply code review fixes and refactor to domain errors
**What changed:**
- `pgstorage/user.go`: PG unique constraint violation (23505) now returns `domain.ErrDuplicateEmail` instead of raw `pgconn.PgError`
- `register.go`: After `CreateUser`, checks for `domain.ErrDuplicateEmail` and returns it directly (handles race condition where pre-check passes but insert fails)
- `register_test.go`: Added `TestRegister_DuplicateEmail_RaceCondition` test case
**Impact:** Concurrent registrations with same email now correctly return `codes.AlreadyExists` instead of `codes.Internal`

### WR-01: Dead code branch in CreateUser — FIXED

**Commit:** (same as CR-01)
**What changed:** The 23505 branch in `CreateUser` now returns `domain.ErrDuplicateEmail` (distinct from the default `return err`), eliminating the dead code path
**Impact:** No more dead code in storage layer

### WR-02: Password length uses byte count instead of character count — FIXED

**Commit:** (same)
**What changed:** `password_validation.go:33` now uses `utf8.RuneCountInString(password)` instead of `len(password)`
**Impact:** Multi-byte Unicode passwords are measured by character count, not byte count

### WR-03: Concrete type for sessionStorage instead of interface — FIXED

**Commit:** (same)
**What changed:**
- `auth_service.go`: `sessionStorage` field type changed from `*redisstorage.RedisStorage` to `SessionStorage` interface
- `NewAuthService` signature updated to accept `SessionStorage` interface
- `bootstrap.go`: `NewAuthService` wrapper updated to pass `authService.SessionStorage`
- Removed `redisstorage` import from `auth_service.go`
**Impact:** AuthService now follows dependency inversion — testable without concrete Redis dependency

## Extra Fixes (beyond review scope)

### EX-01: Domain errors package — auth/internal/domain

All domain errors (`ErrDuplicateEmail`, `ErrInvalidEmail`, `ErrPasswordTooShort`, etc.) and `PasswordValidationError` type moved from `authService` package to `auth/internal/domain`. Services, handlers, and storage import errors from the domain layer.

### EX-02: Test framework migration — minimock + gotest.tools/v3/assert

- Replaced hand-written `mockStorage` with minimock-generated `StorageMock` (auto-verifies expected calls)
- Replaced `testing.T` assertions with `gotest.tools/v3/assert` for clearer failure messages
- Organized register tests as suite with `newRegisterSuite()` setup helper
- Added exact violation count assertion in password validation tests (fixes IN-02)
- Added `TestRegister_StorageError` and `TestRegister_EmailNormalization` test cases

## Info Findings (not in fix scope)

### IN-01: Email validation is minimal — NOT FIXED (accepted per project scope)
### IN-02: Test violation count check — FIXED (as part of EX-02)

## Verification

```
$ cd auth && go test ./internal/services/authService/... -v -count=1
PASS — 35 tests (27 password validation + 8 register)

$ cd auth && go build ./...
OK
```
