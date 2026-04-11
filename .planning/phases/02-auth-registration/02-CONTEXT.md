# Phase 2: Auth Registration - Context

**Gathered:** 2026-04-12
**Status:** Ready for planning

<domain>
## Phase Boundary

Implement user registration via gRPC with comprehensive password validation (length, character classes, sequential/keyboard/repeated character detection) and unit tests covering all rules with boundary cases. Registration persists users to PostgreSQL via the existing PGStorage skeleton.

</domain>

<decisions>
## Implementation Decisions

### Password Validation
- **D-01:** Detect sequential characters across ASCII sequences (abcd, 1234, dcba, 4321), QWERTY keyboard rows (qwertyuiop, asdfghjkl, zxcvbnm), numpad patterns (789, 456, 123), and their reverses — all case-insensitive
- **D-02:** Also reject 4+ identical consecutive characters (aaaa, 1111) — treated as a separate rule alongside sequential detection
- **D-03:** Return ALL failing rules at once, not just the first failure — collect all violations into a structured error
- **D-04:** Password validation lives in `internal/services/authService/password_validation.go` with tests in `password_validation_test.go` — matches CLAUDE.md convention of one file per concern
- **D-05:** Case-insensitive matching for all sequence detection — lowercase before checking

### Registration Response
- **D-06:** Phase 2: Register returns `{user_id, email, created_at}` only. JWT tokens added in Phase 3 when JWT implementation is ready — clean phasing without stubs
- **D-07:** Email validation uses basic format check (contains @, has domain part, no spaces) — not full RFC 5322

### Error Handling
- **D-08:** Password validation errors use per-rule messages — client sees exactly which rules failed (e.g., "password must contain uppercase letter", "contains sequential characters")
- **D-09:** Duplicate email returns explicit AlreadyExists error ("user with this email already exists") — acceptable tradeoff for academic project
- **D-10:** Define separate error types per password rule (ErrPasswordTooShort, ErrMissingUppercase, ErrMissingLowercase, ErrMissingDigit, ErrMissingSpecialChar, ErrSequentialChars, ErrRepeatedChars) and return structured list of violations

### Testing
- **D-11:** Table-driven tests (Go idiomatic) — one TestValidatePassword with test cases table `[{name, password, wantErr, wantRules}]`
- **D-12:** Test BOTH password validation logic AND service-level Register flow (with mocked storage) — covers full registration path
- **D-13:** Comprehensive boundary test cases: 3-char (allowed) vs 4-char (rejected) for EACH sequence type — ASCII ascending, ASCII descending, keyboard rows, keyboard reversed, numpad, repeated chars (~12+ boundary cases)

### Claude's Discretion
- Exact regular expression or algorithm for email format validation
- Internal helper function decomposition within password_validation.go
- SQL query details for user creation (skeleton already has initTables)
- Proto message field naming for RegisterResponse (user_id, email, created_at)
- Mock interface design for storage in register tests

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Architecture & Structure
- `CLAUDE.md` — Full project spec, service structure, Clean Architecture conventions, security rules
- `workspace/01 - Architecture/Overview.md` — System architecture overview
- `workspace/02 - Services/Auth Service.md` — Auth service API and responsibilities

### Decisions
- `workspace/04 - Decisions/ADR Log.md` — Architecture decisions (ADR-005: Clean Architecture)

### Requirements
- `.planning/REQUIREMENTS.md` — AUTH-01, AUTH-02, AUTH-08 requirements for this phase

### Existing Code (Phase 1 skeleton)
- `auth/internal/services/authService/auth_service.go` — AuthService struct with Storage/SessionStorage interfaces (methods TBD)
- `auth/internal/storage/pgstorage/pgstorage.go` — PGStorage with initTables (users table exists)
- `auth/internal/models/models.go` — User model (ID, Email, PasswordHash, CreatedAt, UpdatedAt)
- `auth/internal/api/auth_service_api/register.go` — Register handler (currently Unimplemented stub)
- `auth/api/auth_api/auth_service.proto` — Proto definitions for RegisterRequest/RegisterResponse

### Conventions
- `.planning/codebase/CONVENTIONS.md` — Naming, error handling, code style, DI patterns

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `auth/internal/storage/pgstorage/pgstorage.go` — PGStorage with pool, initTables already creates users table (id, email, password_hash, created_at, updated_at)
- `auth/internal/models/models.go` — User model with all needed fields
- `auth/internal/api/auth_service_api/register.go` — Handler stub ready to be implemented

### Established Patterns
- Clean Architecture: handler → service → repository via interfaces (Phase 1 D-11)
- One file per method in handler/service directories
- gRPC error codes: InvalidArgument for validation, AlreadyExists for duplicates, Internal for unexpected errors
- Structured logging with slog, never log secrets

### Integration Points
- AuthService.storage field (currently `*pgstorage.PGStorage`) needs Storage interface methods: CreateUser, GetUserByEmail
- Register handler calls `api.service.Register(ctx, email, password)` and maps domain errors to gRPC status codes
- Proto RegisterResponse needs to return user_id, email, created_at (tokens deferred to Phase 3)

</code_context>

<specifics>
## Specific Ideas

No specific requirements — all decisions captured in the decisions section above.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 02-auth-registration*
*Context gathered: 2026-04-12*
