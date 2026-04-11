# Testing Patterns

**Analysis Date:** 2026-04-11

## Test Framework

**Runner:**
- `go test` (Go standard testing framework)
- No external test framework required
- Config: Tests discovered automatically in `*_test.go` files

**Assertion Library:**
- No external assertion library (use standard `if` comparisons + `t.Errorf()`)
- Pattern: Explicit error messages for failures

**Run Commands:**
```bash
go test ./...              # Run all tests
go test -v ./...           # Verbose output with test names
go test -run TestName ./...# Run specific test by name
go test -cover ./...       # Show coverage percentage
go test -coverprofile=coverage.out ./...  # Generate coverage profile
go tool cover -html=coverage.out          # View coverage in browser
go test -race ./...        # Run with race detector
go test -timeout 10s ./...  # Set timeout (default 10m)
```

## Test File Organization

**Location:**
- Co-located with implementation: `password_validation_test.go` next to `password_validation.go`
- Tests in same package as code being tested (e.g., `authservice_test` package for `authservice` code)

**Naming:**
- File: `<module>_test.go`
- Test function: `Test<FunctionName>` (e.g., `TestValidatePassword`, `TestRegister`, `TestGenerateToken`)
- Subtests: Use `t.Run("scenario", func(t *testing.T) { ... })` for variations

**Structure:**
```
auth/
├── internal/
│   ├── services/
│   │   └── authservice/
│   │       ├── auth_service.go
│   │       ├── register.go
│   │       ├── login.go
│   │       ├── password_validation.go
│   │       ├── password_validation_test.go    # Tests for password_validation.go
│   │       └── auth_service_test.go           # Tests for auth_service.go (integration-style)
│   ├── storage/
│   │   └── pgstorage/
│   │       ├── pgstorage.go
│   │       ├── user.go
│   │       └── user_test.go                   # Tests for user.go
```

## Test Structure

**Suite Organization:**
```go
// password_validation_test.go
package authservice

import (
    "testing"
)

func TestValidatePassword(t *testing.T) {
    tests := []struct {
        name      string
        password  string
        wantError bool
        errMsg    string
    }{
        {
            name:      "valid password",
            password:  "MyP@ssw0rd123",
            wantError: false,
        },
        {
            name:      "too short",
            password:  "Short1!",
            wantError: true,
            errMsg:    "password must be at least 12 characters",
        },
        {
            name:      "no uppercase",
            password:  "mypassword123!",
            wantError: true,
            errMsg:    "password must contain at least one uppercase letter",
        },
        {
            name:      "sequential digits",
            password:  "MyPass1234!",
            wantError: true,
            errMsg:    "password contains 4 sequential characters",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidatePassword(tt.password)
            if (err != nil) != tt.wantError {
                t.Errorf("ValidatePassword() error = %v, wantError %v", err, tt.wantError)
            }
            if tt.wantError && err != nil && tt.errMsg != "" {
                if err.Error() != tt.errMsg {
                    t.Errorf("ValidatePassword() error message = %q, want %q", err.Error(), tt.errMsg)
                }
            }
        })
    }
}
```

**Patterns:**
- Setup: Initialize fixtures in test function (not shared setup) to keep tests independent
- Teardown: Defer cleanup operations (e.g., `defer db.Close()`)
- Assertion: Use `if ... t.Errorf(...)` pattern, never `t.Fatal()` unless test must stop
- Subtest loop: Table-driven tests with `t.Run()` for organization

**Example with Subtests:**
```go
func TestRegister(t *testing.T) {
    t.Run("successful registration", func(t *testing.T) {
        // Setup
        mockRepo := &mockUserRepository{}
        service := NewAuthService(mockRepo, mockTokenStore, mockLogger)

        // Execute
        user, err := service.Register(context.Background(), "user@example.com", "SecurePass123!")

        // Assert
        if err != nil {
            t.Errorf("Register() unexpected error: %v", err)
        }
        if user == nil {
            t.Errorf("Register() returned nil user")
        }
        if user.Email != "user@example.com" {
            t.Errorf("Register() email = %q, want %q", user.Email, "user@example.com")
        }
    })

    t.Run("duplicate email returns error", func(t *testing.T) {
        // Setup
        mockRepo := &mockUserRepository{
            getByEmailErr: ErrUserAlreadyExists,
        }
        service := NewAuthService(mockRepo, mockTokenStore, mockLogger)

        // Execute
        user, err := service.Register(context.Background(), "existing@example.com", "SecurePass123!")

        // Assert
        if err == nil {
            t.Errorf("Register() expected error for duplicate email, got nil")
        }
        if user != nil {
            t.Errorf("Register() should return nil user on error")
        }
        if !errors.Is(err, ErrUserAlreadyExists) {
            t.Errorf("Register() error = %v, want ErrUserAlreadyExists", err)
        }
    })
}
```

## Mocking

**Framework:** Manual mocks (interfaces + test implementations, no external library)

**Patterns:**
```go
// Define interface in service package (authservice/auth_service.go)
type UserRepository interface {
    GetByEmail(ctx context.Context, email string) (*User, error)
    Create(ctx context.Context, user *User) error
    GetByID(ctx context.Context, id string) (*User, error)
}

// In test file (auth_service_test.go)
type mockUserRepository struct {
    getByEmailFunc func(ctx context.Context, email string) (*User, error)
    createFunc     func(ctx context.Context, user *User) error
    getByIDFunc    func(ctx context.Context, id string) (*User, error)
    
    // Track calls for assertions
    callsGetByEmail int
    callsCreate     int
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
    m.callsGetByEmail++
    if m.getByEmailFunc != nil {
        return m.getByEmailFunc(ctx, email)
    }
    return nil, nil
}

func (m *mockUserRepository) Create(ctx context.Context, user *User) error {
    m.callsCreate++
    if m.createFunc != nil {
        return m.createFunc(ctx, user)
    }
    return nil
}

func (m *mockUserRepository) GetByID(ctx context.Context, id string) (*User, error) {
    if m.getByIDFunc != nil {
        return m.getByIDFunc(ctx, id)
    }
    return nil, nil
}

// Usage in test
func TestRegisterCallsRepository(t *testing.T) {
    mockRepo := &mockUserRepository{
        getByEmailFunc: func(ctx context.Context, email string) (*User, error) {
            return nil, nil // Simulate user doesn't exist
        },
    }
    service := NewAuthService(mockRepo, mockTokenStore, mockLogger)
    
    service.Register(context.Background(), "new@example.com", "SecurePass123!")
    
    if mockRepo.callsGetByEmail != 1 {
        t.Errorf("Register() should call GetByEmail once, called %d times", mockRepo.callsGetByEmail)
    }
}
```

**What to Mock:**
- External dependencies: Database (UserRepository), cache (TokenStore), message queue (KafkaProducer)
- Network calls: Never hit real services in unit tests
- Time-dependent code: Use `time.Now()` injection or freezegun-style mocking
- Random values: Seed deterministically in tests

**What NOT to Mock:**
- Internal business logic: Test the real logic, not a mock of it
- Password validation: Test the actual algorithm
- Shamir arithmetic: Test real GF(256) operations (these are security-critical)
- Error formatting: Test actual error values, not mocked ones

## Fixtures and Factories

**Test Data:**
```go
// In auth_service_test.go - factory functions for test data
func makeTestUser(id, email string) *User {
    return &User{
        ID:        id,
        Email:     email,
        Password:  "hashedpassword123",
        CreatedAt: time.Now(),
    }
}

func makeTestTokenPair() *TokenPair {
    return &TokenPair{
        AccessToken:  "eyJhbGciOiJSUzI1NiJ...",
        RefreshToken: "refresh_token_123",
        ExpiresIn:    900, // 15 minutes
    }
}

func TestLogin(t *testing.T) {
    mockRepo := &mockUserRepository{
        getByEmailFunc: func(ctx context.Context, email string) (*User, error) {
            return makeTestUser("user123", "user@example.com"), nil
        },
    }
    service := NewAuthService(mockRepo, mockTokenStore, mockLogger)
    
    tokens, err := service.Login(context.Background(), "user@example.com", "password")
    // assertions...
}
```

**Location:**
- Test fixtures: In same file as test, as helper functions
- Shared fixtures: In `testdata/` subdirectory if reused across multiple test files
- Database fixtures: Use `fixtures.sql` in `testdata/` or populate via helper functions in tests

**Example fixtures directory:**
```
auth/internal/services/authservice/
├── auth_service.go
├── auth_service_test.go
└── testdata/
    ├── fixtures.sql        # SQL for seeding test database
    └── golden.json         # Expected output for golden tests (if used)
```

## Coverage

**Requirements:** No hard minimum enforced, but aim for:
- All service layer (`internal/services/`): 80%+ coverage
- Handlers (`internal/api/`): 60%+ coverage (mocking calls to services)
- Storage layer (`internal/storage/`): 70%+ (use test database for integration tests)
- Utility functions (password validation, etc.): 90%+

**View Coverage:**
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out        # Opens browser
go tool cover -html=coverage.out -o coverage.html

# Get coverage for specific package
go test -coverprofile=coverage.out ./internal/services/authservice/
go tool cover -func=coverage.out        # Text output by function
```

**Uncovered code to avoid:**
- Main graceful shutdown paths (hard to test, covered by manual testing)
- Initialization code in `main()` (covered by manual testing)
- Error cases in database migrations (test in isolation, not in unit tests)

## Test Types

**Unit Tests:**
- Scope: Single function or method
- Approach: Mock all dependencies
- Location: `*_test.go` file next to implementation
- Example: Test password validation, token generation, error handling
- These should run in <1ms per test

**Integration Tests:**
- Scope: Service layer + real dependencies (test database, redis)
- Approach: Use test containers or in-memory alternatives
- Location: Same package, but marked with `// +build integration` tag (or use `TestIntegration` prefix)
- Example: Test register flow with real database, test token refresh from redis
- Run separately: `go test -tags integration ./...`
- Setup: Use `TestMain(m *testing.M)` to initialize test database

**Example Integration Test Setup:**
```go
// +build integration

package authservice

import (
    "testing"
    "github.com/jackc/pgx/v5/pgxpool"
)

var testDB *pgxpool.Pool

func TestMain(m *testing.M) {
    var err error
    // Connect to test database (e.g., postgres://test:test@localhost:5432/2fa_test)
    testDB, err = pgxpool.New(context.Background(), os.Getenv("TEST_DATABASE_URL"))
    if err != nil {
        panic(err)
    }
    defer testDB.Close()
    
    // Run migrations
    // ...
    
    code := m.Run()
    os.Exit(code)
}

func TestRegisterIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }
    
    // Real storage
    storage := pgstorage.NewPGStorage(testDB)
    service := NewAuthService(storage, mockTokenStore, mockLogger)
    
    user, err := service.Register(context.Background(), "test@example.com", "SecurePass123!")
    if err != nil {
        t.Fatalf("Register() failed: %v", err)
    }
    
    // Verify in database
    retrieved, err := storage.GetByEmail(context.Background(), "test@example.com")
    if err != nil {
        t.Fatalf("GetByEmail() failed: %v", err)
    }
    if retrieved.ID != user.ID {
        t.Errorf("Register() user ID = %q, retrieved %q", user.ID, retrieved.ID)
    }
}
```

**E2E Tests:**
- Not implemented in initial phase
- If added later: Use gRPC client to test full service workflows
- Location: Separate `e2e/` directory
- Tools: Could use `grpcurl` or generated gRPC client

## Common Patterns

**Async Testing:**
```go
func TestRefreshTokenConcurrency(t *testing.T) {
    service := NewAuthService(mockRepo, mockTokenStore, mockLogger)
    
    // Test concurrent token refresh
    done := make(chan bool, 10)
    for i := 0; i < 10; i++ {
        go func() {
            _, err := service.RefreshToken(context.Background(), "refresh_token_123")
            if err != nil {
                t.Errorf("RefreshToken() failed: %v", err)
            }
            done <- true
        }()
    }
    
    // Wait for all goroutines
    for i := 0; i < 10; i++ {
        <-done
    }
}
```

**Context Testing:**
```go
func TestRegisterContextCancellation(t *testing.T) {
    mockRepo := &mockUserRepository{
        createFunc: func(ctx context.Context, user *User) error {
            // Simulate slow operation
            <-ctx.Done()
            return context.Canceled
        },
    }
    service := NewAuthService(mockRepo, mockTokenStore, mockLogger)
    
    ctx, cancel := context.WithCancel(context.Background())
    cancel() // Cancel immediately
    
    _, err := service.Register(ctx, "test@example.com", "SecurePass123!")
    if err == nil {
        t.Errorf("Register() should fail with canceled context")
    }
}
```

**Error Testing:**
```go
func TestRegisterErrorHandling(t *testing.T) {
    tests := []struct {
        name        string
        email       string
        password    string
        repoErr     error
        expectedErr error
    }{
        {
            name:        "invalid email",
            email:       "",
            password:    "SecurePass123!",
            expectedErr: ErrInvalidEmail,
        },
        {
            name:        "invalid password",
            email:       "test@example.com",
            password:    "short",
            expectedErr: ErrInvalidPassword,
        },
        {
            name:        "database error",
            email:       "test@example.com",
            password:    "SecurePass123!",
            repoErr:     ErrDatabase,
            expectedErr: ErrDatabase,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockRepo := &mockUserRepository{
                createFunc: func(ctx context.Context, user *User) error {
                    return tt.repoErr
                },
            }
            service := NewAuthService(mockRepo, mockTokenStore, mockLogger)
            
            _, err := service.Register(context.Background(), tt.email, tt.password)
            if !errors.Is(err, tt.expectedErr) {
                t.Errorf("Register() error = %v, want %v", err, tt.expectedErr)
            }
        })
    }
}
```

**Secrets Protection in Tests:**
- Never hardcode real private keys or TOTP secrets
- Use test keys generated in test setup (e.g., `crypto/rand`)
- Mock crypto operations where testing algorithm is not the goal
- Example: Test JWT token expiry with fake key, not real RSA key

## Security-Critical Testing

**Password Validation Tests:**
- Test all rules: length, character classes, sequence detection
- Test boundary conditions: exactly 11 chars (fail), exactly 12 chars (pass)
- Test sequences: "1234", "abcd", "qwer" and reverse sequences
- Coverage: 100% for this module

**Shamir Secret Sharing Tests:**
- Test split/combine correctness: split 32 bytes, get 3 shares, combine 2 → original
- Test with different thresholds: 2-of-3
- Test share validation: invalid shares should be rejected
- Use deterministic test vectors (known input → known output)
- Coverage: 100% for GF(256) arithmetic

**JWT Token Tests:**
- Test valid token parsing: RS256 with test key
- Test expired token: should be rejected
- Test tampered token: signature verification should fail
- Test token generation: verify claims are set correctly
- Never test with real private keys; generate test keys

**AES-GCM Tests (for MPC nodes):**
- Test encryption/decryption roundtrip
- Test different nonce sizes (16 bytes required for GCM)
- Test authentication: tampered ciphertext should fail
- Test with deterministic nonce (for testing), crypto/rand in production

## Test Isolation

**No Shared State:**
- Each test function is independent
- No `init()` or `var` at package level for test data
- Database: Each test starts fresh (truncate tables or use transactions that rollback)
- Redis: Each test uses unique key prefixes

**Example Cleanup:**
```go
func TestUserRepository(t *testing.T) {
    t.Run("create user", func(t *testing.T) {
        // Setup test database connection
        db := setupTestDB(t)
        defer db.Close() // Cleanup
        
        repo := pgstorage.NewPGStorage(db)
        // Test...
    })
    
    t.Run("get user by email", func(t *testing.T) {
        // Fresh setup for this test
        db := setupTestDB(t)
        defer db.Close()
        
        repo := pgstorage.NewPGStorage(db)
        // Test...
    })
}

func setupTestDB(t *testing.T) *pgxpool.Pool {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    db, err := pgxpool.New(ctx, os.Getenv("TEST_DATABASE_URL"))
    if err != nil {
        t.Fatalf("failed to connect to test database: %v", err)
    }
    
    // Run migrations or truncate tables here
    return db
}
```

---

*Testing analysis: 2026-04-11*
