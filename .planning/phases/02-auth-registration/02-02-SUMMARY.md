---
phase: 02-auth-registration
plan: 02
subsystem: auth
tags: [registration, bcrypt, grpc, clean-architecture, unit-tests]

# Dependency graph
requires:
  - phase: 02-auth-registration
    plan: 01
    provides: ValidatePassword function with PasswordValidationError type
provides:
  - Register method on AuthService with email/password validation and bcrypt hashing
  - Storage interface (CreateUser, GetUserByEmail) for testability
  - gRPC Register handler with proper error code mapping
  - PGStorage CreateUser and GetUserByEmail with parameterized queries
affects: [02-auth-registration plan 03 (login uses GetUserByEmail and bcrypt compare)]

# Tech tracking
tech-stack:
  added: []
  patterns: [Storage interface for DI/testability, mock storage for unit tests, domain error to gRPC code mapping]

key-files:
  created:
    - auth/internal/services/authService/register.go
    - auth/internal/services/authService/register_test.go
    - auth/internal/storage/pgstorage/user.go
  modified:
    - auth/internal/services/authService/auth_service.go
    - auth/internal/api/auth_service_api/register.go
    - auth/internal/bootstrap/bootstrap.go

key-decisions:
  - "AuthService.storage refactored from *pgstorage.PGStorage to Storage interface for unit test mockability"
  - "GetUserByEmail returns (nil, nil) for not-found convention rather than sentinel error"
  - "Email normalized to lowercase with TrimSpace before duplicate check and storage"
  - "gRPC handler excludes PasswordHash from RegisterResponse (SEC-03)"

patterns-established:
  - "Storage interface in service package, concrete implementation in storage package"
  - "Mock storage implementing Storage interface for black-box unit tests"
  - "Domain errors (ErrDuplicateEmail, ErrInvalidEmail) mapped to gRPC codes in handler layer"

requirements-completed: [AUTH-01]

# Metrics
duration: 2min
completed: 2026-04-12
---

# Phase 02 Plan 02: User Registration Flow Summary

**Full registration flow with Storage interface DI, PGStorage parameterized queries, bcrypt cost=12 hashing, gRPC handler error mapping, and 7 mock-based unit tests**

## Performance

- **Duration:** 2 min
- **Started:** 2026-04-12T03:11:28Z
- **Completed:** 2026-04-12T03:13:43Z
- **Tasks:** 2/2
- **Files modified:** 6

## Accomplishments

- Refactored AuthService to use Storage interface instead of concrete PGStorage for testability
- Implemented CreateUser and GetUserByEmail on PGStorage with parameterized queries ($1-$5) preventing SQL injection
- Register method validates email format, validates password strength, checks for duplicates, hashes with bcrypt cost=12, generates UUID, persists user
- gRPC Register handler maps PasswordValidationError to InvalidArgument, ErrDuplicateEmail to AlreadyExists, internal errors masked (SEC-02)
- Response returns user data with Tokens: nil (D-06), never exposes PasswordHash (SEC-03)
- 7 table-driven test cases with mock storage covering success, invalid email (4 variants), weak password, and duplicate email scenarios

## Task Commits

Each task was committed atomically:

1. **Task 1: Refactor AuthService to interface, add Storage methods and Register service method** - `177e783` (feat)
2. **Task 2: Wire Register gRPC handler and create service-level Register tests** - `1d03907` (feat)

## Files Created/Modified

- `auth/internal/services/authService/auth_service.go` - Storage interface with CreateUser/GetUserByEmail, ErrDuplicateEmail, ErrInvalidEmail
- `auth/internal/services/authService/register.go` - Register method, validateEmail helper, COST_BCRYPT=12
- `auth/internal/services/authService/register_test.go` - 7 table-driven tests with mockStorage
- `auth/internal/storage/pgstorage/user.go` - CreateUser and GetUserByEmail with parameterized queries
- `auth/internal/api/auth_service_api/register.go` - gRPC handler with domain-to-gRPC error mapping
- `auth/internal/bootstrap/bootstrap.go` - Updated to accept Storage interface

## Decisions Made

- AuthService.storage refactored from concrete *pgstorage.PGStorage to Storage interface for unit test mockability
- GetUserByEmail returns (nil, nil) for not-found rather than a sentinel error, simplifying caller logic
- Email normalized to lowercase with TrimSpace before both duplicate check and storage
- gRPC handler excludes PasswordHash from RegisterResponse per SEC-03

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Storage interface ready for Login flow (GetUserByEmail already implemented)
- bcrypt hashing in place for password comparison in Login
- gRPC error mapping pattern established for reuse in Login/Logout handlers

---
*Phase: 02-auth-registration*
*Completed: 2026-04-12*
