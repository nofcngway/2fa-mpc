# Phase 2: Auth Registration - Research

**Researched:** 2026-04-12
**Domain:** Go gRPC service implementation, password validation, bcrypt hashing, PostgreSQL persistence
**Confidence:** HIGH

## Summary

Phase 2 implements user registration for the Auth service. The existing Phase 1 skeleton provides the gRPC server, PGStorage with `users` table, AuthService struct, and a Register handler stub returning `Unimplemented`. This phase fills in the registration flow: password validation with comprehensive rules (length, character classes, sequential/repeated detection), bcrypt hashing, user creation in PostgreSQL, and thorough unit tests.

The core complexity is in password validation -- detecting sequential characters across ASCII sequences, QWERTY keyboard rows, numpad patterns, and repeated characters, all case-insensitive. Decision D-03 requires returning ALL failing rules at once, not just the first. The proto RegisterResponse currently includes a `TokenPair` field, but D-06 says to return only `{user_id, email, created_at}` in Phase 2 -- the tokens field will be nil until Phase 3.

**Primary recommendation:** Implement password validation as a pure function in its own file with table-driven tests, then wire CreateUser storage method and Register handler. Add `golang.org/x/crypto` and `github.com/google/uuid` to go.mod.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Detect sequential characters across ASCII sequences (abcd, 1234, dcba, 4321), QWERTY keyboard rows (qwertyuiop, asdfghjkl, zxcvbnm), numpad patterns (789, 456, 123), and their reverses -- all case-insensitive
- **D-02:** Also reject 4+ identical consecutive characters (aaaa, 1111) -- treated as a separate rule alongside sequential detection
- **D-03:** Return ALL failing rules at once, not just the first failure -- collect all violations into a structured error
- **D-04:** Password validation lives in `internal/services/authService/password_validation.go` with tests in `password_validation_test.go`
- **D-05:** Case-insensitive matching for all sequence detection -- lowercase before checking
- **D-06:** Phase 2: Register returns `{user_id, email, created_at}` only. JWT tokens added in Phase 3 -- tokens field in RegisterResponse is nil
- **D-07:** Email validation uses basic format check (contains @, has domain part, no spaces) -- not full RFC 5322
- **D-08:** Password validation errors use per-rule messages -- client sees exactly which rules failed
- **D-09:** Duplicate email returns explicit AlreadyExists error
- **D-10:** Define separate error types per password rule (ErrPasswordTooShort, ErrMissingUppercase, ErrMissingLowercase, ErrMissingDigit, ErrMissingSpecialChar, ErrSequentialChars, ErrRepeatedChars) and return structured list of violations
- **D-11:** Table-driven tests (Go idiomatic) -- one TestValidatePassword with test cases table
- **D-12:** Test BOTH password validation logic AND service-level Register flow (with mocked storage)
- **D-13:** Comprehensive boundary test cases: 3-char vs 4-char for EACH sequence type (~12+ boundary cases)

### Claude's Discretion
- Exact regular expression or algorithm for email format validation
- Internal helper function decomposition within password_validation.go
- SQL query details for user creation (skeleton already has initTables)
- Proto message field naming for RegisterResponse (user_id, email, created_at)
- Mock interface design for storage in register tests

### Deferred Ideas (OUT OF SCOPE)
None
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| AUTH-01 | User can register with email and password (bcrypt cost=12) | Existing PGStorage with users table, User model, Register handler stub. Need: CreateUser method, bcrypt hashing via golang.org/x/crypto, UUID generation |
| AUTH-02 | Password validated before hashing -- min 12 chars, 1 lowercase, 1 uppercase, 1 digit, 1 special char, no 4+ sequential chars | Pure validation function in password_validation.go, sequence detection across ASCII/QWERTY/numpad, structured multi-error return |
| AUTH-08 | Password validation has unit tests covering each rule and boundary cases (3 vs 4 sequential chars) | Table-driven tests in password_validation_test.go, service-level Register test with mocked storage |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| golang.org/x/crypto | v0.50.0 | bcrypt password hashing | Official Go crypto extension, bcrypt.GenerateFromPassword / bcrypt.CompareHashAndPassword [VERIFIED: `go list -m golang.org/x/crypto@latest`] |
| github.com/google/uuid | v1.6.0 | UUID generation for user IDs | Standard Go UUID library, uuid.New() returns v4 UUID [VERIFIED: `go list -m github.com/google/uuid@latest`] |
| github.com/jackc/pgx/v5 | v5.9.1 | PostgreSQL driver (already in go.mod) | Already used in Phase 1 PGStorage [VERIFIED: go.mod] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| testing (stdlib) | built-in | Unit testing framework | All test files, table-driven tests [VERIFIED: stdlib] |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| golang.org/x/crypto/bcrypt | argon2 | bcrypt is explicitly required by CLAUDE.md (cost=12), no choice here |
| github.com/google/uuid | crypto/rand + manual format | uuid package handles RFC 4122 correctly, no reason to hand-roll |

**Installation:**
```bash
cd auth && go get golang.org/x/crypto@v0.50.0 github.com/google/uuid@v1.6.0
```

## Architecture Patterns

### Files to Create/Modify

```
auth/
├── internal/
│   ├── services/authService/
│   │   ├── auth_service.go          # MODIFY: add Register method, update Storage interface
│   │   ├── password_validation.go   # CREATE: ValidatePassword + helpers
│   │   ├── password_validation_test.go # CREATE: table-driven tests
│   │   └── register.go             # CREATE: Register method (service layer)
│   ├── storage/pgstorage/
│   │   └── user.go                  # CREATE: CreateUser, GetUserByEmail
│   └── api/auth_service_api/
│       └── register.go              # MODIFY: implement handler
```

### Pattern 1: Multi-Error Password Validation
**What:** ValidatePassword returns a slice of all violated rules, not just first failure
**When to use:** Whenever all validation rules must be reported simultaneously (D-03, D-10)
**Example:**
```go
// Source: CONTEXT.md D-03, D-10
type PasswordValidationError struct {
    Violations []error
}

func (e *PasswordValidationError) Error() string {
    // Join all violation messages
}

var (
    ErrPasswordTooShort   = errors.New("password must be at least 12 characters")
    ErrMissingUppercase   = errors.New("password must contain at least one uppercase letter")
    ErrMissingLowercase   = errors.New("password must contain at least one lowercase letter")
    ErrMissingDigit       = errors.New("password must contain at least one digit")
    ErrMissingSpecialChar = errors.New("password must contain at least one special character")
    ErrSequentialChars    = errors.New("password must not contain 4 or more sequential characters")
    ErrRepeatedChars      = errors.New("password must not contain 4 or more identical consecutive characters")
)

func ValidatePassword(password string) error {
    var violations []error
    // Check all rules, append to violations
    if len(violations) > 0 {
        return &PasswordValidationError{Violations: violations}
    }
    return nil
}
```

### Pattern 2: Sequential Character Detection
**What:** Sliding window (size 4) checks if characters form consecutive ASCII values, keyboard row subsequences, or numpad patterns
**When to use:** D-01 requires checking ASCII, QWERTY, numpad sequences and their reverses
**Example:**
```go
// Source: CONTEXT.md D-01, D-05
// Predefined sequence strings (all lowercase per D-05)
var sequences = []string{
    "abcdefghijklmnopqrstuvwxyz",  // ASCII letters
    "0123456789",                   // ASCII digits
    "qwertyuiop",                   // QWERTY row 1
    "asdfghjkl",                    // QWERTY row 2
    "zxcvbnm",                      // QWERTY row 3
    "7894561230",                   // Numpad (rows: 789, 456, 123, 0)
}

func containsSequential(password string, minLen int) bool {
    lower := strings.ToLower(password)
    for _, seq := range sequences {
        if containsSubseqOfLength(lower, seq, minLen) {
            return true
        }
        // Check reverse
        if containsSubseqOfLength(lower, reverse(seq), minLen) {
            return true
        }
    }
    return false
}
```

### Pattern 3: Handler-to-Service Error Mapping
**What:** Handler receives domain errors and maps to gRPC status codes
**When to use:** Register handler converts password validation errors to InvalidArgument, duplicate email to AlreadyExists
**Example:**
```go
// Source: CLAUDE.md error handling conventions
func (api *AuthServiceAPI) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
    user, err := api.service.Register(ctx, req.Email, req.Password)
    if err != nil {
        var validErr *authService.PasswordValidationError
        if errors.As(err, &validErr) {
            return nil, status.Error(codes.InvalidArgument, validErr.Error())
        }
        if errors.Is(err, authService.ErrDuplicateEmail) {
            return nil, status.Error(codes.AlreadyExists, "user with this email already exists")
        }
        return nil, status.Error(codes.Internal, "internal error")
    }
    return &pb.RegisterResponse{
        User: &models.User{
            Id:        user.ID,
            Email:     user.Email,
            CreatedAt: user.CreatedAt.Format(time.RFC3339),
        },
    }, nil
}
```

### Pattern 4: Storage Interface for Testability (D-12)
**What:** AuthService depends on Storage interface, not concrete PGStorage
**When to use:** Enables mocking in register_test.go without a database
**Example:**
```go
// Source: CLAUDE.md Clean Architecture, CONTEXT.md D-12
type Storage interface {
    CreateUser(ctx context.Context, user *models.User) error
    GetUserByEmail(ctx context.Context, email string) (*models.User, error)
}
```
**Note:** The current AuthService struct uses `*pgstorage.PGStorage` directly. Phase 2 must refactor the `storage` field to use the `Storage` interface to enable mocking. This is a required change for D-12.

### Anti-Patterns to Avoid
- **Returning first error only:** D-03 explicitly requires ALL violations at once
- **Hardcoding sequence lists inline:** Extract to package-level vars for readability and testability
- **Logging passwords:** CLAUDE.md explicitly forbids logging secrets -- never log the password argument
- **Returning internal error details to client:** gRPC errors should be sanitized (SEC-02)

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Password hashing | Custom hash function | golang.org/x/crypto/bcrypt | Timing-safe comparison, salt management, cost factor built-in |
| UUID generation | crypto/rand + manual formatting | github.com/google/uuid | RFC 4122 compliance, proper v4 generation |
| PostgreSQL connection pooling | Manual connection management | pgxpool (already in PGStorage) | Connection lifecycle, health checks, pool sizing handled |

**Key insight:** The only hand-rolled crypto in this phase is the password validation logic (not cryptographic). bcrypt and UUID are standard library concerns.

## Common Pitfalls

### Pitfall 1: Sequence Detection Off-by-One
**What goes wrong:** Checking for 3 consecutive characters instead of 4, or vice versa
**Why it happens:** Sliding window boundary confusion -- window of size 4 means `i` goes to `len(s)-4`, not `len(s)-3`
**How to avoid:** Test boundary cases explicitly: "abc" (3 chars, ALLOWED) vs "abcd" (4 chars, REJECTED). D-13 requires ~12+ such boundary cases
**Warning signs:** Tests pass with wrong threshold

### Pitfall 2: Numpad Sequence Representation
**What goes wrong:** Treating numpad as simple "123456789" when actual layout is rows 789/456/123/0
**Why it happens:** Confusion between ASCII digit order and numpad physical layout
**How to avoid:** Define numpad sequences per actual keyboard layout. D-01 specifies numpad patterns (789, 456, 123). Consider: is "7894" sequential on numpad? (789 row then 4 on next row) -- if treating "789456123" as one continuous sequence, "7894" would be 4 consecutive in that string
**Warning signs:** Legitimate passwords rejected or numpad sequences allowed

### Pitfall 3: bcrypt Cost Timing
**What goes wrong:** bcrypt cost=12 takes ~250ms per hash, tests with many bcrypt calls become slow
**Why it happens:** Cost=12 is production security, not test speed
**How to avoid:** Password validation tests don't need bcrypt (they test validation logic). Only the Register service test calls bcrypt, and only 1-2 times. If testing becomes slow, that's acceptable -- don't reduce cost in production code
**Warning signs:** Test suite taking >10 seconds

### Pitfall 4: AuthService Concrete vs Interface Dependency
**What goes wrong:** AuthService currently uses `*pgstorage.PGStorage` directly -- cannot mock storage for unit tests
**Why it happens:** Phase 1 scaffold used concrete types
**How to avoid:** Change `storage` field to `Storage` interface type. Update `NewAuthService` to accept interface. PGStorage already satisfies it implicitly once methods are added
**Warning signs:** Cannot write unit tests for Register without a database

### Pitfall 5: Proto RegisterResponse Includes TokenPair
**What goes wrong:** Current proto has `tokens` field in RegisterResponse. Phase 2 should return only user data (D-06)
**Why it happens:** Proto was designed for the final flow where Register returns tokens
**How to avoid:** Leave proto unchanged (tokens field stays). In Phase 2, set `Tokens: nil` in the response. Phase 3 will populate it. Avoids proto regeneration churn.
**Warning signs:** Client code expecting tokens to be present

### Pitfall 6: UNIQUE Constraint Error Detection
**What goes wrong:** PostgreSQL returns a generic error for UNIQUE violations -- need to detect and convert to domain error
**Why it happens:** pgx wraps errors; need to unwrap to get the PG error code
**How to avoid:** Check for pgx error code 23505 (unique_violation). pgx v5 provides `pgconn.PgError` with code field
**Warning signs:** Duplicate email returns Internal instead of AlreadyExists
```go
// [VERIFIED: pgx v5 error handling pattern]
import "github.com/jackc/pgx/v5/pgconn"

var pgErr *pgconn.PgError
if errors.As(err, &pgErr) && pgErr.Code == "23505" {
    return ErrDuplicateEmail
}
```

## Code Examples

### bcrypt Usage
```go
// Source: golang.org/x/crypto/bcrypt [ASSUMED - standard API]
import "golang.org/x/crypto/bcrypt"

const COST_BCRYPT = 12

// Hash password
hash, err := bcrypt.GenerateFromPassword([]byte(password), COST_BCRYPT)
if err != nil {
    return fmt.Errorf("hash password: %w", err)
}
user.PasswordHash = string(hash)
```

### UUID Generation
```go
// Source: github.com/google/uuid [ASSUMED - standard API]
import "github.com/google/uuid"

user := &models.User{
    ID:    uuid.New().String(),
    Email: email,
}
```

### CreateUser SQL
```go
// Source: existing pgstorage pattern from Phase 1
func (ps *PGStorage) CreateUser(ctx context.Context, user *models.User) error {
    _, err := ps.pool.Exec(ctx, `
        INSERT INTO users (id, email, password_hash, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5)
    `, user.ID, user.Email, user.PasswordHash, user.CreatedAt, user.UpdatedAt)
    return err
}
```

### GetUserByEmail SQL
```go
func (ps *PGStorage) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
    var user models.User
    err := ps.pool.QueryRow(ctx, `
        SELECT id, email, password_hash, created_at, updated_at
        FROM users WHERE email = $1
    `, email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, nil // user not found
        }
        return nil, err
    }
    return &user, nil
}
```

### Table-Driven Test Pattern (D-11)
```go
// Source: Go testing conventions, CONTEXT.md D-11, D-13
func TestValidatePassword(t *testing.T) {
    tests := []struct {
        name      string
        password  string
        wantErr   bool
        wantRules []error // expected violation types
    }{
        {"valid strong password", "MyStr0ng!Pass99", false, nil},
        {"too short", "Sh0rt!Pass", true, []error{ErrPasswordTooShort}},
        {"missing uppercase", "alllowercase1!", true, []error{ErrMissingUppercase}},
        // boundary: 3 sequential ASCII (allowed)
        {"3 sequential ascii allowed", "MyP@ssw0rdabc", false, nil},
        // boundary: 4 sequential ASCII (rejected)
        {"4 sequential ascii rejected", "MyP@ssw0rdabcd", true, []error{ErrSequentialChars}},
        // ... ~12+ boundary cases per D-13
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidatePassword(tt.password)
            // assert err and wantRules
        })
    }
}
```

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none needed (Go built-in) |
| Quick run command | `cd auth && go test ./internal/services/authService/ -v -count=1` |
| Full suite command | `cd auth && go test ./... -v -count=1` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| AUTH-01 | Register creates user in PG with bcrypt hash | unit (mocked storage) | `cd auth && go test ./internal/services/authService/ -run TestRegister -v` | Wave 0 |
| AUTH-02 | Password validation rejects weak passwords | unit | `cd auth && go test ./internal/services/authService/ -run TestValidatePassword -v` | Wave 0 |
| AUTH-08 | Boundary cases for 3 vs 4 sequential chars | unit | `cd auth && go test ./internal/services/authService/ -run TestValidatePassword -v` | Wave 0 |

### Sampling Rate
- **Per task commit:** `cd auth && go test ./internal/services/authService/ -v -count=1`
- **Per wave merge:** `cd auth && go test ./... -v -count=1`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `auth/internal/services/authService/password_validation_test.go` -- covers AUTH-02, AUTH-08
- [ ] `auth/internal/services/authService/register_test.go` -- covers AUTH-01 (mocked storage)
- [ ] `golang.org/x/crypto` dependency: `cd auth && go get golang.org/x/crypto@v0.50.0`
- [ ] `github.com/google/uuid` dependency: `cd auth && go get github.com/google/uuid@v1.6.0`

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | bcrypt cost=12, password complexity rules per CLAUDE.md |
| V3 Session Management | no | Deferred to Phase 3 (JWT/refresh tokens) |
| V4 Access Control | no | No authorization in registration |
| V5 Input Validation | yes | Password validation (length, chars, sequences), email format check |
| V6 Cryptography | yes | bcrypt via golang.org/x/crypto (never hand-roll) |

### Known Threat Patterns for Auth Registration

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Weak password storage | Information Disclosure | bcrypt cost=12, never store plaintext |
| Email enumeration via Register | Information Disclosure | D-09 accepts explicit AlreadyExists for academic project (acknowledged tradeoff) |
| SQL injection in CreateUser | Tampering | Parameterized queries via pgx ($1, $2) |
| Password logging | Information Disclosure | CLAUDE.md: NEVER log passwords -- enforce in code review |
| Credential stuffing | Spoofing | Rate limiting deferred to Gateway (v2), password complexity helps |

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| MD5/SHA password hashing | bcrypt/argon2 | 2010s | bcrypt with cost=12 is required by CLAUDE.md |
| Return first validation error | Return all errors at once | UX best practice | D-03 requires multi-error return |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | bcrypt.GenerateFromPassword API unchanged in x/crypto v0.50.0 | Code Examples | LOW -- API stable for 10+ years |
| A2 | uuid.New().String() returns v4 UUID string | Code Examples | LOW -- API stable |
| A3 | pgconn.PgError.Code == "23505" for unique violations in pgx v5 | Pitfall 6 | MEDIUM -- need to verify error type path |

## Open Questions

1. **Numpad sequence boundaries**
   - What we know: D-01 specifies numpad patterns (789, 456, 123)
   - What's unclear: Should "7894" be treated as 4 consecutive in numpad? (crossing rows 789->456). If numpad is defined as "789456123" continuous string, then yes
   - Recommendation: Treat numpad as continuous string "7894561230" -- simplest interpretation, catches cross-row sequences. This matches the sequence detection approach for QWERTY rows

2. **AuthService interface refactoring scope**
   - What we know: Current AuthService.storage is `*pgstorage.PGStorage` (concrete). D-12 requires mocked storage tests
   - What's unclear: Should we also refactor SessionStorage to interface in Phase 2, or only what's needed?
   - Recommendation: Only refactor `storage` field to `Storage` interface. Leave `sessionStorage` as concrete `*redisstorage.RedisStorage` until Phase 3 needs it

3. **Proto modification vs nil tokens**
   - What we know: D-06 says return only user data in Phase 2. Current proto has TokenPair field
   - What's unclear: Modify proto to remove tokens, or leave tokens nil?
   - Recommendation: Leave proto unchanged, return `Tokens: nil`. Avoids proto regeneration and keeps forward compatibility for Phase 3

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go compiler | All code | Yes | 1.26.2 | -- |
| golang.org/x/crypto | bcrypt (AUTH-01) | Not in go.mod yet | v0.50.0 (latest) | Must add via go get |
| github.com/google/uuid | User ID generation | Not in go.mod yet | v1.6.0 (latest) | Must add via go get |
| PostgreSQL | User persistence | Via docker-compose | -- | Required, no fallback |
| pgx/v5 | DB driver | In go.mod | v5.9.1 | -- |

**Missing dependencies with no fallback:**
- `golang.org/x/crypto` and `github.com/google/uuid` must be added to go.mod before implementation

**Missing dependencies with fallback:**
- None -- all dependencies are addable via `go get`

## Sources

### Primary (HIGH confidence)
- Codebase inspection: auth/internal/* -- all existing code read directly
- go.mod -- verified dependency versions
- `go list -m golang.org/x/crypto@latest` -- confirmed v0.50.0
- `go list -m github.com/google/uuid@latest` -- confirmed v1.6.0

### Secondary (MEDIUM confidence)
- CLAUDE.md -- project conventions and security requirements
- 02-CONTEXT.md -- locked decisions from discuss phase

### Tertiary (LOW confidence)
- bcrypt/uuid API details -- standard knowledge, flagged as [ASSUMED]

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- dependencies verified via go list, existing code inspected
- Architecture: HIGH -- Phase 1 skeleton establishes clear patterns, decisions are locked
- Pitfalls: HIGH -- based on direct code inspection (concrete vs interface, proto tokens field, PG error codes)

**Research date:** 2026-04-12
**Valid until:** 2026-05-12 (stable Go ecosystem, locked decisions)
