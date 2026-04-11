# Coding Conventions

**Analysis Date:** 2026-04-11

## Naming Patterns

**Files:**
- **Handler files** (gRPC API): One method per file within `internal/api/<service>_service_api/`. Example: `register.go`, `login.go`, `refresh_token.go` (snake_case)
- **Service files** (Business logic): One method per file within `internal/services/<serviceName>/`. Example: `setup.go`, `verify.go`, `disable.go` (snake_case)
- **Storage files**: One entity type per file within `internal/storage/pgstorage/`. Example: `user.go`, `session.go`, `share.go` (snake_case)
- **Test files**: Append `_test.go` suffix to the implementation file (same directory). Example: `password_validation_test.go` in `internal/services/authService/`
- **Utility/Domain files**: Clear, single-responsibility naming. Example: `shamir.go`, `gf256.go`, `aes.go`, `totp.go` (snake_case)

**Functions:**
- **Handler functions**: Receiver method on service struct. Example: `(api *AuthServiceAPI) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.TokenPair, error)`
- **Service methods**: Receiver method on service struct. Example: `(s *AuthService) ValidatePassword(password string) error`
- **Repository methods**: Receiver method on storage struct. Example: `(ps *PGStorage) CreateUser(ctx context.Context, user *User) error`
- **Internal helpers**: PascalCase (unexported). Example: `validateEmail`, `hashPassword`, `generateJWT`
- **Exported functions**: PascalCase. Example: `NewAuthService`, `NewPGStorage`, `Split`, `Combine`

**Variables:**
- **Short-lived loop/temp variables**: Single letter (i, j, ctx, err). Example: `for i := 0; i < len(shares); i++`
- **Domain objects**: Clear descriptive names. Example: `user`, `share`, `token`, `secret`, `nonce`
- **Constants**: SCREAMING_SNAKE_CASE. Example: `MAX_ATTEMPTS`, `RATE_LIMIT_WINDOW`, `JWT_EXPIRY_MINUTES`, `COST_BCRYPT`
- **Interfaces**: Capitalized, descriptive. Example: `Storage`, `Service`, `Encryptor`, `Producer`
- **Error variables**: Prefixed with `err` or `Err`. Example: `errNotFound`, `ErrInvalidPassword`

**Types:**
- **Struct names**: PascalCase, nouns. Example: `User`, `AuthService`, `PGStorage`, `Share`, `TokenPair`
- **Interface names**: PascalCase, -er suffix for behavior. Example: `Encryptor`, `Producer`, `Validator`
- **Proto message names**: PascalCase, noun. Example: `RegisterRequest`, `User`, `TokenPair`

## Code Style

**Formatting:**
- **Tool**: gofmt (built-in Go formatter)
- **Indentation**: 1 tab = 8 spaces (Go standard)
- **Line length**: No hard limit, but keep under 120 characters for readability
- **Blank lines**: Use between logical sections within functions, between methods

**Linting:**
- **Tool**: golangci-lint (or equivalent)
- **Key rules enforced**:
  - Unused variables/imports → error
  - Unhandled errors → warning
  - Snake_case for package names, camelCase for identifiers
  - Exported functions/constants require comments

**Comments:**
- **Linting enforced**: Every exported function/type must have a comment
- **Format**: Start with the exported name. Example:
  ```go
  // AuthService handles user authentication operations.
  type AuthService struct { ... }
  
  // Register creates a new user account with email and password validation.
  func (s *AuthService) Register(ctx context.Context, email, password string) error { ... }
  ```
- **Unexported helpers**: Optional but recommended. Example:
  ```go
  // validatePassword checks password against policy rules.
  func validatePassword(password string) error { ... }
  ```
- **Complex logic blocks**: Inline comments explain intent, not what code does. Example:
  ```go
  // Verify the OTP within ±1 time window (RFC 6238)
  if !totp.Verify(secret, code, time.Now()) { ... }
  ```

## Import Organization

**Order** (gofmt enforces this automatically):
1. Standard library imports (e.g., `context`, `fmt`, `time`)
2. Third-party imports (e.g., `google.golang.org/grpc`, `github.com/jackc/pgx/v5`)
3. Local/project imports (e.g., `auth/internal/models`, `auth/internal/api`)

**Path Aliases:**
- **No aliases** — use full import paths with package prefixes
- **Local packages**: Import as `auth/internal/models` (not aliased)
- **Example**:
  ```go
  import (
    "context"
    "time"
    
    "google.golang.org/grpc"
    "github.com/jackc/pgx/v5"
    
    "auth/internal/models"
    "auth/internal/services/authService"
  )
  ```

## Error Handling

**Patterns:**
- **Explicit error checks**: Always check returned errors immediately
  ```go
  user, err := s.storage.GetByEmail(ctx, email)
  if err != nil {
    return fmt.Errorf("failed to fetch user: %w", err)
  }
  ```
- **Wrapping errors**: Use `%w` with `fmt.Errorf` to preserve error chain
- **gRPC errors**: Convert domain errors to gRPC status codes in handlers
  - `codes.InvalidArgument` — validation failed (bad password, invalid email)
  - `codes.NotFound` — resource not found (user doesn't exist)
  - `codes.Unauthenticated` — authentication failed (wrong password, invalid token)
  - `codes.AlreadyExists` — resource already exists (duplicate email)
  - `codes.Internal` — unexpected error (database failure)
  
  **Example in handler** (`internal/api/auth_service_api/register.go`):
  ```go
  func (api *AuthServiceAPI) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.TokenPair, error) {
    token, err := api.service.Register(ctx, req.Email, req.Password)
    if err != nil {
      if errors.Is(err, authService.ErrInvalidPassword) {
        return nil, status.Error(codes.InvalidArgument, "password does not meet policy")
      }
      if errors.Is(err, authService.ErrEmailExists) {
        return nil, status.Error(codes.AlreadyExists, "user with this email already exists")
      }
      // Log unexpected error
      slog.ErrorContext(ctx, "register failed", "error", err)
      return nil, status.Error(codes.Internal, "failed to register user")
    }
    return token, nil
  }
  ```

## Logging

**Framework:** `log/slog` (Go 1.21+ standard library)

**Patterns:**
- **Structured logging only**: All logs use key-value pairs
  ```go
  slog.InfoContext(ctx, "user registered", "user_id", userID, "email", email)
  slog.ErrorContext(ctx, "password validation failed", "reason", "too short", "length", len(password))
  ```
- **Secret data NEVER logged**: No passwords, TOTP secrets, share data, encryption keys, JWT tokens
  - ✗ `slog.Debug("password hash", "hash", hash)` (password material exposed)
  - ✗ `slog.Debug("share data", "share", hexEncode(shareBytes))` (secret exposed)
  - ✓ `slog.Info("user password reset", "user_id", userID)`
  - ✓ `slog.Info("share stored", "user_id", userID, "share_index", 1)`
- **Error logging**: Always include error with context
  ```go
  slog.ErrorContext(ctx, "database query failed", "operation", "CreateUser", "error", err)
  ```
- **Metric context**: Include observable metadata (user_id, operation, status)
  ```go
  slog.InfoContext(ctx, "2fa verification", "user_id", userID, "status", "success")
  ```

## Dependency Injection

**Pattern**: Bootstrap factory functions in `internal/bootstrap/` create all dependencies

**Example structure** (`internal/bootstrap/auth_service.go`):
```go
// AuthService creates and returns a configured AuthService.
func AuthService(cfg *config.Config, storage Storage, redisClient *redis.Client, kafkaProducer *kafka.Writer) *authService.AuthService {
  return &authService.AuthService{
    storage: storage,
    redis: redisClient,
    producer: kafkaProducer,
    // ... other dependencies
  }
}
```

**Main.go pattern** (`cmd/app/main.go`):
```go
func main() {
  cfg := config.Load()
  
  // Bootstrap dependencies
  storage := bootstrap.PGStorage(cfg)
  redisClient := bootstrap.Redis(cfg)
  kafkaProducer := bootstrap.KafkaProducer(cfg)
  
  authService := bootstrap.AuthService(cfg, storage, redisClient, kafkaProducer)
  authAPI := bootstrap.AuthServiceAPI(cfg, authService)
  grpcServer := bootstrap.GRPCServer(cfg, authAPI)
  
  // Start and shutdown handlers...
}
```

## Configuration

**Format**: YAML (`config.yaml` in service root)

**Loading**: `config/config.go` uses `gopkg.in/yaml.v3`

**Structure**:
```go
type Config struct {
  Server struct {
    Port int `yaml:"port"`
  } `yaml:"server"`
  Database struct {
    DSN string `yaml:"dsn"`
  } `yaml:"database"`
  Redis struct {
    Addr string `yaml:"addr"`
  } `yaml:"redis"`
  Kafka struct {
    Brokers []string `yaml:"brokers"`
  } `yaml:"kafka"`
  JWT struct {
    AccessExpiry  int    `yaml:"access_expiry_minutes"`
    RefreshExpiry int    `yaml:"refresh_expiry_days"`
    PrivateKey    string `yaml:"private_key_path"`
    PublicKey     string `yaml:"public_key_path"`
  } `yaml:"jwt"`
}

func Load() *Config {
  data, err := os.ReadFile("config.yaml")
  if err != nil {
    panic(err)
  }
  var cfg Config
  if err := yaml.Unmarshal(data, &cfg); err != nil {
    panic(err)
  }
  return &cfg
}
```

## Repository (Storage) Pattern

**Location**: `internal/storage/<type>/`

**Structure** (`internal/storage/pgstorage/user.go`):
```go
// User operations
func (ps *PGStorage) CreateUser(ctx context.Context, user *models.User) error {
  query := `INSERT INTO users (id, email, password_hash, created_at) VALUES ($1, $2, $3, $4)`
  _, err := ps.pool.Exec(ctx, query, user.ID, user.Email, user.PasswordHash, user.CreatedAt)
  return err
}

func (ps *PGStorage) GetByEmail(ctx context.Context, email string) (*models.User, error) {
  user := &models.User{}
  query := `SELECT id, email, password_hash, created_at FROM users WHERE email = $1`
  err := ps.pool.QueryRow(ctx, query, email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)
  if err == pgx.ErrNoRows {
    return nil, ErrNotFound
  }
  return user, err
}
```

**Key patterns**:
- One method per CRUD operation or query
- Always accept `ctx context.Context` as first parameter
- Return errors directly (convert to gRPC codes in handler layer)
- Use parameterized queries ($1, $2, etc.) to prevent SQL injection

## Handler (gRPC API) Pattern

**Location**: `internal/api/<service>_service_api/`

**Structure** (`internal/api/auth_service_api/register.go`):
```go
func (api *AuthServiceAPI) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.TokenPair, error) {
  // Validate request
  if req.Email == "" || req.Password == "" {
    return nil, status.Error(codes.InvalidArgument, "email and password required")
  }
  
  // Call service
  token, err := api.service.Register(ctx, req.Email, req.Password)
  if err != nil {
    // Convert domain errors to gRPC codes
    if errors.Is(err, authService.ErrInvalidPassword) {
      return nil, status.Error(codes.InvalidArgument, "password does not meet policy")
    }
    // Log unexpected errors only
    slog.ErrorContext(ctx, "register failed", "error", err)
    return nil, status.Error(codes.Internal, "failed to register user")
  }
  
  return token, nil
}
```

**Key patterns**:
- Minimal validation in handler (basic checks only)
- Delegate business logic to service layer
- Convert errors to appropriate gRPC status codes
- No detailed error messages returned to client for security (log internally instead)

## Service (Business Logic) Pattern

**Location**: `internal/services/<serviceName>/`

**Structure** (`internal/services/authService/register.go`):
```go
func (s *AuthService) Register(ctx context.Context, email, password string) (*models.TokenPair, error) {
  // Validate password
  if err := s.validatePassword(password); err != nil {
    return nil, ErrInvalidPassword
  }
  
  // Check email uniqueness
  _, err := s.storage.GetByEmail(ctx, email)
  if err == nil {
    return nil, ErrEmailExists
  }
  if !errors.Is(err, pgstorage.ErrNotFound) {
    return nil, fmt.Errorf("failed to check email: %w", err)
  }
  
  // Hash password
  hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
  if err != nil {
    return nil, fmt.Errorf("password hashing failed: %w", err)
  }
  
  // Create user
  user := &models.User{
    ID: uuid.New(),
    Email: email,
    PasswordHash: string(hash),
    CreatedAt: time.Now(),
  }
  if err := s.storage.CreateUser(ctx, user); err != nil {
    return nil, fmt.Errorf("failed to create user: %w", err)
  }
  
  // Generate tokens
  token, err := s.generateTokens(ctx, user.ID)
  if err != nil {
    return nil, fmt.Errorf("token generation failed: %w", err)
  }
  
  // Publish event
  if err := s.producer.PublishUserRegistered(ctx, user.ID, email); err != nil {
    slog.WarnContext(ctx, "failed to publish event", "error", err)
    // Don't fail the operation for event publishing failure
  }
  
  return token, nil
}

// validatePassword checks password policy rules (unexported helper).
func (s *AuthService) validatePassword(password string) error {
  if len(password) < 12 {
    return ErrInvalidPassword
  }
  // ... additional validation logic
  return nil
}
```

**Key patterns**:
- Contains all business logic and validation
- Uses repository for data access
- Returns domain errors (not gRPC codes)
- Publishes events asynchronously (failures don't block main operation)

## Crypto and Security

**Password hashing**:
- Algorithm: bcrypt with cost=12
- Always pre-validate password before hashing (see `internal/services/authService/password_validation.go`)
- Example:
  ```go
  if err := validatePassword(password); err != nil {
    return err // ErrInvalidPassword
  }
  hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
  ```

**JWT (RS256)**:
- Private key: Held only by Auth Service
- Public key: Distributed to Gateway and other services for verification
- Claims structure:
  ```go
  type Claims struct {
    UserID string `json:"user_id"`
    jwt.RegisteredClaims
  }
  ```
- Expiry: Access 15 minutes, Refresh 7 days

**Secret handling (TOTP, Shamir shares)**:
- NEVER persist whole secrets
- After split/combine in memory → immediately zeroize using `subtle.ConstantTimeCompare` or manual byte clearing
- Example:
  ```go
  secret := generateTOTPSecret() // 20 bytes
  shares := shamir.Split(secret, 3, 2)
  // Clear secret from memory
  for i := range secret {
    secret[i] = 0
  }
  // Now only shares (pieces of secret) are used
  ```

**AES-256-GCM encryption** (at-rest for shares):
- Unique nonce per operation: `crypto/rand` (12 bytes)
- Encryption key: From config (NODE_ID specific)
- Example in MPC Node:
  ```go
  func (s *ShareService) Encrypt(plaintext []byte, key []byte) (ciphertext, nonce []byte, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
      return nil, nil, err
    }
    gcm, err := cipher.NewGCM(block)
    if err != nil {
      return nil, nil, err
    }
    nonce := make([]byte, gcm.NonceSize())
    if _, err := rand.Read(nonce); err != nil {
      return nil, nil, err
    }
    ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
    return ciphertext, nonce, nil
  }
  ```

## Function Design

**Size**: 
- Aim for functions under 30 lines
- Extract helper functions for complex logic blocks
- Each function should have a single responsibility

**Parameters**:
- Max 3-4 parameters; use struct for related params
- Always include `ctx context.Context` as first parameter for handlers/services
- Example:
  ```go
  // Good: related params grouped
  func (s *TwoFAService) Setup(ctx context.Context, userID string) (string, error) { ... }
  
  // Avoid: too many params
  func Store(ctx context.Context, userID, shareIndex, nodeID, nonce string, data []byte) error { ... }
  // Better: group in struct
  func (s *ShareService) Store(ctx context.Context, req *StoreRequest) error { ... }
  ```

**Return Values**:
- Always return errors as last value: `(result Type, error)`
- Prefer `(value, error)` over `error` only
- Never ignore returned errors in production code

## Module Design

**Exports**:
- Capitalize only what's meant for external use
- Private types/functions use lowercase (package-internal)
- Example:
  ```go
  // Exported
  type User struct { ... }
  func NewUser(email string) *User { ... }
  
  // Private
  type userRepository interface { ... }
  func validatePassword(pwd string) error { ... }
  ```

**Barrel Files** (No barrel files used in this project):
- Each file imports directly from subpackages
- Example: `import "auth/internal/services/authService"` (not from `auth/internal/services/`)

## Middleware and Interceptors

**Location**: `internal/middleware/interceptors.go`

**gRPC Interceptors** (unary):
```go
func LoggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
  slog.InfoContext(ctx, "grpc call", "method", info.FullMethod)
  
  start := time.Now()
  resp, err := handler(ctx, req)
  duration := time.Since(start)
  
  if err != nil {
    slog.ErrorContext(ctx, "grpc error", "method", info.FullMethod, "error", err, "duration_ms", duration.Milliseconds())
  } else {
    slog.InfoContext(ctx, "grpc success", "method", info.FullMethod, "duration_ms", duration.Milliseconds())
  }
  
  return resp, err
}
```

**Registration in gRPC server**:
```go
grpcServer := grpc.NewServer(
  grpc.UnaryInterceptor(LoggingInterceptor),
)
```

---

*Convention analysis: 2026-04-11*
