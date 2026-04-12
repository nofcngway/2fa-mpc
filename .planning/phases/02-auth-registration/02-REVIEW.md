---
phase: 02-auth-registration
reviewed: 2026-04-12T12:00:00Z
depth: standard
files_reviewed: 8
files_reviewed_list:
  - auth/internal/services/authService/password_validation.go
  - auth/internal/services/authService/password_validation_test.go
  - auth/internal/services/authService/register.go
  - auth/internal/services/authService/register_test.go
  - auth/internal/storage/pgstorage/user.go
  - auth/internal/services/authService/auth_service.go
  - auth/internal/api/auth_service_api/register.go
  - auth/internal/bootstrap/bootstrap.go
findings:
  critical: 1
  warning: 3
  info: 2
  total: 6
status: issues_found
---

# Phase 02: Code Review Report

**Reviewed:** 2026-04-12T12:00:00Z
**Depth:** standard
**Files Reviewed:** 8
**Status:** issues_found

## Summary

The auth registration phase implements user registration with password validation, email validation, bcrypt hashing, and PostgreSQL persistence. The code follows Clean Architecture conventions well, with proper layered separation and gRPC error mapping. However, there is a critical race condition in duplicate email detection, a dead-code branch in the storage layer, and a design inconsistency where a concrete Redis type is used instead of the declared interface.

## Critical Issues

### CR-01: Race condition in duplicate email detection — concurrent registrations can bypass uniqueness check

**File:** `auth/internal/services/authService/register.go:32-38` and `auth/internal/storage/pgstorage/user.go:14-27`
**Issue:** The registration flow performs a read-then-write pattern: `GetUserByEmail` checks for duplicates (line 32), then `CreateUser` inserts (line 56). Between these two calls, another concurrent request can insert the same email. When this happens, PostgreSQL raises a unique constraint violation (code 23505), but `CreateUser` in pgstorage returns the raw `pgconn.PgError` without mapping it to `ErrDuplicateEmail`. The gRPC handler then returns `codes.Internal` instead of `codes.AlreadyExists`, leaking an internal error to the client.
**Fix:** Map the unique constraint violation in `CreateUser` to a domain-level sentinel error, and handle it in the service layer:

```go
// pgstorage/user.go
import "github.com/vbncursed/vkr/auth/internal/models"

var ErrDuplicateKey = errors.New("duplicate key")

func (ps *PGStorage) CreateUser(ctx context.Context, user *models.User) error {
    _, err := ps.pool.Exec(ctx, `
        INSERT INTO users (id, email, password_hash, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5)
    `, user.ID, user.Email, user.PasswordHash, user.CreatedAt, user.UpdatedAt)
    if err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" {
            return ErrDuplicateKey
        }
        return err
    }
    return nil
}
```

Then in `register.go`, check for `pgstorage.ErrDuplicateKey` after `CreateUser` and return `ErrDuplicateEmail`.

## Warnings

### WR-01: Dead code branch in CreateUser — unique constraint handling has no distinct behavior

**File:** `auth/internal/storage/pgstorage/user.go:19-26`
**Issue:** The `if` block on lines 20-22 checks for PostgreSQL error code 23505 (unique violation) but then returns the same `err` as the else branch on line 24. Both paths return the raw error, making the unique constraint check a no-op. This is likely a leftover from incomplete implementation.
**Fix:** Either map to a domain error (see CR-01) or remove the dead branch until it is needed:

```go
if err != nil {
    return err
}
```

### WR-02: Password length validation uses byte count instead of character count

**File:** `auth/internal/services/authService/password_validation.go:65`
**Issue:** `len(password)` returns byte count, not rune/character count. A password with multi-byte Unicode characters (e.g., accented letters, CJK) could pass the 12-character minimum while being fewer than 12 actual characters, or fail it while having 12+ characters. The `containsSubseq` function (line 133-134) also slices by byte index, which could produce invalid substrings for multi-byte input.
**Fix:** Use `utf8.RuneCountInString(password)` for the length check. For the sequence detection, the current approach works because all sequences are ASCII-only and `strings.ToLower` preserves byte positions for ASCII characters, so this is lower severity for the sequence checks. For length:

```go
import "unicode/utf8"

if utf8.RuneCountInString(password) < MIN_PASSWORD_LENGTH {
    violations = append(violations, ErrPasswordTooShort)
}
```

### WR-03: Concrete type used for sessionStorage instead of interface

**File:** `auth/internal/services/authService/auth_service.go:32`
**Issue:** The `AuthService` struct uses `*redisstorage.RedisStorage` as the type for `sessionStorage`, despite a `SessionStorage` interface being defined on line 24. This violates dependency inversion — the service layer directly depends on a concrete infrastructure type. This makes testing harder (tests must pass `nil` or a real `*redisstorage.RedisStorage`) and breaks the Clean Architecture pattern established by the `Storage` interface.
**Fix:** Use the `SessionStorage` interface as the field type (once methods are added in Phase 3):

```go
type AuthService struct {
    storage        Storage
    sessionStorage SessionStorage
}

func NewAuthService(storage Storage, sessionStorage SessionStorage) *AuthService {
    return &AuthService{
        storage:        storage,
        sessionStorage: sessionStorage,
    }
}
```

## Info

### IN-01: Email validation is minimal — accepts structurally invalid addresses

**File:** `auth/internal/services/authService/register.go:64-83`
**Issue:** The `validateEmail` function only checks for `@`, non-empty parts, and a dot in the domain. It accepts addresses like `user@.com`, `user@com.`, `@user@example.com` (multiple @-signs rejected), and domains with consecutive dots. While the project constraints say not to add features outside the scope, this basic validation may allow obviously invalid emails to be stored.
**Fix:** Consider adding checks for leading/trailing dots in domain, consecutive dots, and minimum domain length. Alternatively, use `net/mail.ParseAddress` from the standard library for a more robust check without adding dependencies.

### IN-02: Test coverage does not verify the exact count of violations in multi-error cases

**File:** `auth/internal/services/authService/password_validation_test.go:186-197`
**Issue:** The multi-error test case ("short and missing classes") checks that specific violations are present, but does not verify that no unexpected additional violations are included. The test uses a subset check (`found` loop) rather than an exact match. This means if a code change accidentally adds spurious violations, the test would still pass.
**Fix:** Add a count check:

```go
if len(validationErr.Violations) != len(tt.wantRules) {
    t.Errorf("expected %d violations, got %d: %v",
        len(tt.wantRules), len(validationErr.Violations), validationErr.Violations)
}
```

---

_Reviewed: 2026-04-12T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
