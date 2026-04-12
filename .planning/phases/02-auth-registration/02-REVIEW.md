---
phase: 02-auth-registration
reviewed: 2026-04-12T14:30:00Z
depth: standard
files_reviewed: 9
files_reviewed_list:
  - auth/internal/domain/errors.go
  - auth/internal/services/authService/password_validation.go
  - auth/internal/services/authService/password_validation_test.go
  - auth/internal/services/authService/register.go
  - auth/internal/services/authService/register_test.go
  - auth/internal/storage/pgstorage/user.go
  - auth/internal/services/authService/auth_service.go
  - auth/internal/api/auth_service_api/register.go
  - auth/internal/bootstrap/bootstrap.go
findings:
  critical: 0
  warning: 0
  info: 1
  total: 1
status: issues_found
---

# Phase 02: Code Review Report (Re-review)

**Reviewed:** 2026-04-12T14:30:00Z
**Depth:** standard
**Files Reviewed:** 9
**Status:** issues_found

## Summary

This is a post-fix re-review of the auth registration phase. All four previously reported issues have been correctly resolved:

- **CR-01 (race condition):** Fixed. `register.go:58-63` now checks for `domain.ErrDuplicateEmail` returned by `CreateUser` after a concurrent insert hits the unique constraint. `pgstorage/user.go:22` correctly maps PostgreSQL error code `23505` to `domain.ErrDuplicateEmail`. Test `TestRegister_DuplicateEmail_RaceCondition` validates this path.
- **WR-01 (dead code):** Fixed. The unique constraint check in `pgstorage/user.go:20-24` now maps to `domain.ErrDuplicateEmail` instead of returning the raw error, making the branch meaningful.
- **WR-02 (byte count vs rune count):** Fixed. `password_validation.go:32` now uses `utf8.RuneCountInString(password)` for length validation.
- **WR-03 (concrete Redis type):** Fixed. `auth_service.go:23-24` declares `sessionStorage` as the `SessionStorage` interface type, and `NewAuthService` on line 29 accepts `SessionStorage`. Proper dependency inversion is maintained.

Additionally, the previous IN-02 (test coverage for violation count) has been addressed: `register_test.go` and `password_validation_test.go` now check `len(validationErr.Violations) == len(tt.wantRules)` at line 222, ensuring exact violation count matching.

The codebase is clean and well-structured. Code follows Clean Architecture conventions, proper gRPC error mapping, parameterized SQL queries, and correct error propagation patterns. One minor informational note remains.

## Info

### IN-01: Byte-level sliding window on potentially multi-byte strings

**File:** `auth/internal/services/authService/password_validation.go:100`
**Issue:** `containsSubseq` and `containsRepeated` operate on byte indices (`password[i : i+winSize]` at line 100, `lower[i+j]` at line 119) after `strings.ToLower()`. For passwords containing multi-byte UTF-8 characters (e.g., Cyrillic, CJK), this slices mid-rune. In practice this cannot produce false positives for `containsSubseq` since the reference sequences are ASCII-only and a mid-rune byte slice will never match an ASCII sequence. For `containsRepeated`, certain multi-byte characters with repeated internal bytes could theoretically trigger a false positive, though this is extremely unlikely in real passwords.
**Fix:** No action required for current scope. If internationalized passwords are supported in the future, convert to `[]rune` before sliding window operations, similar to how `reverse()` already does on line 133.

---

_Reviewed: 2026-04-12T14:30:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
