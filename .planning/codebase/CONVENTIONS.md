# Coding Conventions

**Analysis Date:** 2026-04-11

## Naming Patterns

**Files:**
- Source files: `snake_case.go` (e.g., `register.go`, `auth_service.go`, `password_validation.go`)
- Test files: `<module>_test.go` (e.g., `password_validation_test.go`, `auth_service_test.go`)
- One file per logical operation: handlers use one file per RPC method (`register.go`, `login.go`, `refresh_token.go`), services likewise
- Proto files: `snake_case.proto` in `api/` subdirectories (`auth.proto`, `auth_model.proto`)

**Packages:**
- `main` in `cmd/app/`
- Service packages: `authservice` (no spaces, lowercase)
- Handler packages: Named by service layer (e.g., `auth_service_api`)
- Storage packages: Named by storage type (`pgstorage`, `redisstorage`)

**Functions:**
- Public: `PascalCase` (e.g., `Register`, `NewAuthService`, `GetUserByEmail`)
- Private: `camelCase` (e.g., `validatePassword`, `hashPassword`, `generateTokenPair`)
- Handler functions: Named after RPC method (`Register`, `Login`, `RefreshToken`)
- Factory functions: `NewServiceName` pattern (e.g., `NewAuthService`, `NewPGStorage`)

**Variables:**
- Constants: `SCREAMING_SNAKE_CASE` for package-level constants (e.g., `DEFAULT_JWT_EXPIRY`, `BCRYPT_COST`)
- Functions: `camelCase` (e.g., `userID`, `sessionToken`, `passwordHash`)
- Unexported: `camelCase` (e.g., `storage`, `logger`, `kafkaProducer`)
- Interfaces: `PascalCase` ending with `er` or full name (e.g., `UserRepository`, `TokenStore`, `PasswordValidator`)

**Types/Structs:**
- `PascalCase` (e.g., `User`, `Session`, `AuthService`)
- Interface fields: Unexported, accessed via methods
- Receiver names: Short, typically first letter(s) of type (`u *User`, `s *Session`, `as *AuthService`)

## Code Style

**Formatting:**
- Use `gofmt` for all Go code (enforced by `go fmt ./...`)
- Line length: No hard limit, but keep under 120 characters where practical
- Indentation: Use tabs (Go standard)

**Linting:**
- Use `golangci-lint` with configuration for this project (if `.golangci.yml` exists)
- Tools expected: `goimports` (imports ordering), `vet` (standard checks), `errcheck` (error handling)
- Commands:
  ```bash
  go fmt ./...           # Format code
  go vet ./...           # Vet all code
  golangci-lint run      # Full lint (if configured)
  ```

## Import Organization

**Order:**
1. Standard library imports (`fmt`, `crypto`, `errors`, `log/slog`)
2. Google/external packages (`google.golang.org/grpc`, `github.com/jackc/pgx/v5`)
3. Project-relative imports (`github.com/mpc-2fa/auth/internal/...`)
4. Separated by blank lines

**Example:**
```go
import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mpc-2fa/auth/internal/models"
	"github.com/mpc-2fa/auth/internal/services/authservice"
)
```

**Path Aliases:**
- Use `pb` for protobuf-generated code: `import pb "github.com/mpc-2fa/auth/internal/pb"`
- Use service aliases for clarity: `import authasvc "github.com/mpc-2fa/auth/internal/services/authservice"`

## Error Handling

**Patterns:**
- All functions that can fail must return `error` as last return value
- Use `errors.New()` for simple errors, `fmt.Errorf()` for formatted errors with context
- Wrap errors with context: `fmt.Errorf("failed to hash password: %w", err)` (use `%w` to preserve stack)
- Do NOT use `panic()` except in init/main startup (e.g., config loading failure)
- Return `nil` for no error

**gRPC-specific:**
- Convert all domain errors to gRPC status codes before returning from handlers
- Use `status.Error(codes.X, "message")` to wrap errors for gRPC
- Common mappings:
  - Invalid input → `codes.InvalidArgument`
  - Not found → `codes.NotFound`
  - Already exists → `codes.AlreadyExists`
  - Unauthenticated (bad token) → `codes.Unauthenticated`
  - Permission denied → `codes.PermissionDenied`
  - Internal failures → `codes.Internal`

**Example:**
```go
// In service layer
if user == nil {
    return nil, fmt.Errorf("user not found: %w", ErrUserNotFound)
}

// In handler layer (convert to gRPC)
user, err := as.authService.GetUserByEmail(ctx, email)
if err != nil {
    if errors.Is(err, ErrUserNotFound) {
        return nil, status.Error(codes.NotFound, "user not found")
    }
    return nil, status.Error(codes.Internal, "failed to retrieve user")
}
```

## Logging

**Framework:** `log/slog` (Go standard library, structured logging)

**Rules:**
- NEVER log secrets: passwords, TOTP secrets, private keys, JWT tokens, share data, symmetric keys
- Always include request context: user_id, operation name, timestamp (implicit in slog)
- Use structured fields: `slog.String("user_email", email)`, `slog.Int("user_id", userID)`
- Log levels:
  - `Debug`: Development-only, detailed flow information
  - `Info`: Important state transitions (user registered, login successful)
  - `Warn`: Recoverable issues (failed validation attempt, rate limit triggered)
  - `Error`: Error conditions that need attention (database error, crypto failure)

**Patterns:**
```go
// At service initialization
logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

// During operations
logger.InfoContext(ctx, "user registered",
    slog.String("email", user.Email),
    slog.String("user_id", user.ID),
)

logger.WarnContext(ctx, "password validation failed",
    slog.String("user_id", userID),
    slog.String("reason", "too short"),
)

logger.ErrorContext(ctx, "database error",
    slog.String("user_id", userID),
    slog.String("operation", "insert_user"),
    slog.String("error", err.Error()),
)
```

## Comments

**When to Comment:**
- Public types and functions: Always use documentation comments (format: `// TypeName describes...`)
- Complex algorithms: Explain why, not what (code shows what)
- Non-obvious design decisions: "Why does this check exist?"
- Security-sensitive code: Explain the security implication
- Workarounds: If accepting a limitation, document why

**Documentation Comments (GoDoc):**
```go
// Register creates a new user account with email and password.
// Password must pass validation (see ValidatePassword).
// Returns ErrUserAlreadyExists if email is already registered.
func (as *AuthService) Register(ctx context.Context, email, password string) (*User, error)
```

**Inline Comments:**
```go
// Validate password before hashing to fail fast
if err := validatePassword(password); err != nil {
    return fmt.Errorf("password validation failed: %w", err)
}

// Bcrypt cost must be ≥12 per security requirements (ADR-005)
hash, err := bcrypt.GenerateFromPassword([]byte(password), BCRYPT_COST)
```

**No Comments For:**
- Variable names that are clear: `userID` doesn't need `// user's unique identifier`
- Obvious loops: `for i := 0; i < len(users); i++` doesn't need explanation
- Self-documenting code: Good naming removes need for comments

## Function Design

**Size:** Keep functions under 50 lines when possible. If larger, consider extracting helper functions.

**Parameters:**
- Max 3-4 parameters per function
- For related parameters, use a struct (`type RegisterRequest struct { Email, Password string }`)
- Pass `context.Context` as first parameter in functions that accept it
- Use interfaces for dependencies, not concrete types

**Return Values:**
- Error as last return value (Go standard)
- Multiple returns: max 2 meaningful values + error
- Use named returns only for clarity in exported functions
- Wrap errors with context, not just `err`

**Example:**
```go
// Good: clear parameters, error handling
func (s *AuthService) Login(ctx context.Context, email, password string) (*TokenPair, error) {
    if email == "" {
        return nil, fmt.Errorf("email required")
    }
    if password == "" {
        return nil, fmt.Errorf("password required")
    }
    
    user, err := s.userRepo.GetByEmail(ctx, email)
    if err != nil {
        return nil, fmt.Errorf("failed to retrieve user: %w", err)
    }
    if user == nil {
        return nil, ErrUserNotFound
    }
    
    return s.createTokenPair(ctx, user.ID)
}

// Internal helper: validates without context overhead
func (s *AuthService) createTokenPair(ctx context.Context, userID string) (*TokenPair, error) {
    // token generation logic
}
```

## Module Design

**Exports:**
- Only export what other packages need
- Unexported fields accessed via getter methods
- Constructors always exported (`New*` pattern)
- Interfaces always exported

**Example:**
```go
// User model: exported fields for simplicity if immutable
type User struct {
    ID        string
    Email     string
    Password  string // hash, never plaintext
    CreatedAt time.Time
}

// Service: unexported storage, exported constructor
type AuthService struct {
    userRepo   UserRepository  // unexported
    tokenStore TokenStore      // unexported
    logger     *slog.Logger    // unexported
}

func NewAuthService(userRepo UserRepository, tokenStore TokenStore) *AuthService {
    return &AuthService{
        userRepo:   userRepo,
        tokenStore: tokenStore,
        logger:     slog.Default(),
    }
}
```

**Interfaces:**
- Define interfaces in the package that uses them (not the implementation)
- Keep interfaces minimal (single responsibility)
- Name interfaces with `-er` suffix if they describe a single action

**Example in service package:**
```go
// In internal/services/authservice/auth_service.go
type UserRepository interface {
    GetByEmail(ctx context.Context, email string) (*User, error)
    Create(ctx context.Context, user *User) error
    GetByID(ctx context.Context, id string) (*User, error)
}

type TokenStore interface {
    Set(ctx context.Context, key string, value string, ttl time.Duration) error
    Get(ctx context.Context, key string) (string, error)
    Delete(ctx context.Context, key string) error
}
```

**Barrel Files:**
- Use in `api/` for proto imports
- Use in `internal/` only if consolidating related types for clarity
- Example: `internal/models/models.go` exports `User`, `Session`, `TokenPair`

## Security-Specific Conventions

**Password Handling:**
- Always validate before hashing: `validatePassword(password)` returns error if invalid
- Bcrypt cost: Always use `BCRYPT_COST = 12` constant
- Never log plaintext passwords, only operation results
- Zero password strings after use when possible: use `copy(password, make([]byte, len(password)))`

**Token Handling:**
- JWT tokens: RS256 algorithm, access 15min, refresh 7 days (constants in config)
- Access tokens: Never persisted, only in memory for request
- Refresh tokens: Stored in Redis with TTL, invalidated on logout
- Never log full tokens, only token type/ID

**Crypto Operations:**
- Nonce generation: Always use `crypto/rand.Read()` for AES-GCM nonces
- Key management: Load from environment or secure config, never hardcoded
- Error on crypto failures: Never silently fail
- For Shamir: Validate operations return correct types, bounds

**Rate Limiting:**
- Track failed attempts by user_id in Redis
- Key format: `rate_limit:2fa:verify:{user_id}`
- Max 5 failures per 5 minutes per user

## Dependency Injection Pattern

**Bootstrap Pattern:**
- Every dependency created in `internal/bootstrap/` factories
- Factory functions: `NewServiceName(deps...) ServiceInterface`
- Bootstrap receives config, returns initialized service with all wired dependencies
- No global state or singletons

**Example:**
```go
// internal/bootstrap/auth_service.go
func NewAuthService(cfg *config.Config, storage *pgstorage.PGStorage, redisStore redisstorage.TokenStore, logger *slog.Logger) (*authservice.AuthService, error) {
    service := authservice.NewAuthService(storage, redisStore)
    service.SetLogger(logger)
    return service, nil
}

// main.go
cfg, _ := config.Load("config.yaml")
pgStorage, _ := bootstrap.NewPGStorage(cfg.Database)
redisStore, _ := bootstrap.NewRedisTokenStore(cfg.Redis)
authService, _ := bootstrap.NewAuthService(cfg, pgStorage, redisStore, slog.Default())
```

---

*Convention analysis: 2026-04-11*
