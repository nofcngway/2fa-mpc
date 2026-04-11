# Testing Patterns

**Analysis Date:** 2026-04-11

## Test Framework

**Runner:**
- **Framework**: `testing` (Go built-in standard library)
- **Config**: No separate config file — test discovery is automatic
- **Test discovery**: Any file matching `*_test.go` in the same package is discovered automatically
- **Execution**: `go test ./...` or `go test -v ./...` with verbose output

**Assertion Library:**
- **Approach**: Manual assertions using `if`, `reflect.DeepEqual()`, or custom helpers
- **No external assertion libraries** (keeping dependencies minimal)
- **Example**:
  ```go
  if err != nil {
    t.Errorf("expected no error, got %v", err)
  }
  
  if !reflect.DeepEqual(result, expected) {
    t.Errorf("expected %v, got %v", expected, result)
  }
  ```

**Run Commands:**
```bash
go test ./...              # Run all tests in all packages
go test -v ./...           # Run all tests with verbose output
go test -race ./...        # Run all tests with race detector enabled
go test -cover ./...       # Run all tests and show coverage summary
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out  # Generate HTML coverage report
go test ./internal/services/authService/...  # Run tests in specific package
go test -run TestPasswordValidation ./...    # Run only tests matching pattern
go test -timeout 30s ./... # Run tests with 30-second timeout
```

## Test File Organization

**Location:**
- **Co-located pattern** — test files live in the same package as implementation
- **Example structure**:
  ```
  internal/services/authService/
  ├── auth_service.go
  ├── register.go
  ├── login.go
  ├── password_validation.go
  └── password_validation_test.go    # test for password_validation.go
  ```

**Naming:**
- `<implementation>_test.go` (e.g., `password_validation_test.go`)
- Test functions: `Test<FunctionName>` (e.g., `TestValidatePassword`, `TestValidatePasswordInvalidLength`)
- Subtests: `t.Run("description", func(t *testing.T) { ... })`

**Structure:**
```
internal/
├── services/
│   └── authService/
│       ├── password_validation.go
│       ├── password_validation_test.go   # Tests for password validation rules
│       ├── shamir/
│       │   ├── shamir.go
│       │   └── shamir_test.go            # Tests for Shamir split/combine
│       └── totp/
│           ├── totp.go
│           └── totp_test.go              # Tests for TOTP generation/validation
├── storage/
│   └── pgstorage/
│       ├── user.go
│       ├── session.go
│       └── ... (repository tests would go here if using integration testing)
└── crypto/
    ├── aes.go
    └── aes_test.go                       # Tests for AES encryption/decryption
```

## Test Structure

**Suite Organization:**
```go
// password_validation_test.go
package authService

import "testing"

func TestValidatePassword(t *testing.T) {
  tests := []struct {
    name      string
    password  string
    wantError bool
    reason    string
  }{
    {
      name:      "valid_password",
      password:  "SecureP@ss123",
      wantError: false,
    },
    {
      name:      "too_short",
      password:  "Short@1",
      wantError: true,
      reason:    "less than 12 characters",
    },
    {
      name:      "no_uppercase",
      password:  "securepass@ss1",
      wantError: true,
      reason:    "missing uppercase letter",
    },
    // ... more test cases
  }
  
  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      err := ValidatePassword(tt.password)
      if (err != nil) != tt.wantError {
        t.Errorf("ValidatePassword() error = %v, wantError %v", err, tt.wantError)
      }
    })
  }
}
```

**Patterns:**
- **Setup/Teardown**: Use `func setup()` and `func teardown()` for test-wide setup
  ```go
  func setup() *TestContext {
    // Initialize test database, mocks, etc.
    return &TestContext{...}
  }
  
  func TestSomething(t *testing.T) {
    ctx := setup()
    defer ctx.teardown()
    
    // Test code
  }
  ```
- **Table-driven tests**: Use struct slice for multiple test cases (see example above)
- **Assertions**: Use helper functions for common checks
  ```go
  func assertError(t *testing.T, got error, want error) {
    if !errors.Is(got, want) {
      t.Errorf("expected %v, got %v", want, got)
    }
  }
  
  func assertEqual(t *testing.T, got, want interface{}) {
    if !reflect.DeepEqual(got, want) {
      t.Errorf("expected %v, got %v", want, got)
    }
  }
  ```

## Unit Tests

**Scope**: Single function/method in isolation

**Approach**:
- **Mock external dependencies**: Interfaces are mocked in-memory
- **No database access**: Use in-memory doubles or pass mock implementations
- **Fast execution**: Milliseconds per test
- **Deterministic**: No randomness, controlled inputs

**Example** (`internal/services/authService/password_validation_test.go`):
```go
package authService

import "testing"

func TestValidatePassword_MinimumLength(t *testing.T) {
  tests := []struct {
    password string
    wantErr  bool
  }{
    {"Secure@Pass1", false},  // 12 chars — valid
    {"Secur@Pass1", true},     // 11 chars — too short
    {"SecureP@ss1", false},    // 12 chars exactly — valid
  }
  
  for _, tt := range tests {
    err := ValidatePassword(tt.password)
    if (err != nil) != tt.wantErr {
      t.Errorf("password %q: error = %v, wantErr %v", tt.password, err != nil, tt.wantErr)
    }
  }
}

func TestValidatePassword_SequenceDetection(t *testing.T) {
  tests := []struct {
    password string
    wantErr  bool
    desc     string
  }{
    {"Pass@1234word", true, "numeric sequence 1234"},
    {"Pass@1233word", false, "numeric sequence 3 chars"},
    {"Pass@abcdword", true, "alphabetic sequence abcd"},
    {"Pass@abcword", false, "alphabetic sequence 3 chars"},
    {"Qwer@Pass1234", true, "keyboard sequence qwer"},
  }
  
  for _, tt := range tests {
    err := ValidatePassword(tt.password)
    if (err != nil) != tt.wantErr {
      t.Errorf("%s: password %q error = %v, wantErr %v", tt.desc, tt.password, err, tt.wantErr)
    }
  }
}
```

**Mocking Pattern** (for services with dependencies):
```go
// In password_validation_test.go or separate mock file
type mockStorage struct {
  calls map[string]int
  users map[string]*User
}

func (m *mockStorage) GetByEmail(ctx context.Context, email string) (*User, error) {
  m.calls["GetByEmail"]++
  if user, ok := m.users[email]; ok {
    return user, nil
  }
  return nil, ErrNotFound
}

func TestRegister_DuplicateEmail(t *testing.T) {
  mock := &mockStorage{
    calls: make(map[string]int),
    users: map[string]*User{
      "existing@example.com": {Email: "existing@example.com"},
    },
  }
  
  svc := &AuthService{storage: mock}
  _, err := svc.Register(context.Background(), "existing@example.com", "ValidP@ss123")
  
  if err != ErrEmailExists {
    t.Errorf("expected ErrEmailExists, got %v", err)
  }
  if mock.calls["GetByEmail"] != 1 {
    t.Errorf("expected GetByEmail to be called once, was called %d times", mock.calls["GetByEmail"])
  }
}
```

## Integration Tests (Limited)

**Scope**: Multiple components working together (e.g., service + repository)

**Approach**:
- **Real PostgreSQL test database**: Use `docker-compose.test.yaml` with PostgreSQL container
- **Setup/teardown**: Spin up container before tests, clean up after
- **Not the primary testing method** — focus on unit tests for speed

**Example pattern** (if needed):
```go
// database_test.go (only if doing database integration testing)
func TestGetUserIntegration(t *testing.T) {
  if testing.Short() {
    t.Skip("skipping integration test")
  }
  
  // Setup: connect to test database
  pool := setupTestDB(t)
  defer pool.Close()
  
  storage := NewPGStorage(pool)
  
  // Test
  user := &User{ID: uuid.New(), Email: "test@example.com", ...}
  err := storage.CreateUser(context.Background(), user)
  if err != nil {
    t.Fatalf("CreateUser failed: %v", err)
  }
  
  retrieved, err := storage.GetByEmail(context.Background(), "test@example.com")
  if err != nil {
    t.Fatalf("GetByEmail failed: %v", err)
  }
  
  if retrieved.ID != user.ID {
    t.Errorf("expected ID %v, got %v", user.ID, retrieved.ID)
  }
}
```

## Testing Cryptographic Functions

**Shamir Secret Sharing Tests** (`internal/services/twofaService/shamir/shamir_test.go`):

```go
package shamir

import (
  "bytes"
  "testing"
)

func TestSplit_Basic(t *testing.T) {
  secret := []byte("supersecretdata")
  n := 3    // total shares
  threshold := 2  // minimum shares to recover
  
  shares := Split(secret, n, threshold)
  
  if len(shares) != n {
    t.Errorf("expected %d shares, got %d", n, len(shares))
  }
  
  for i, share := range shares {
    if share == nil || len(share) == 0 {
      t.Errorf("share %d is empty", i)
    }
  }
}

func TestSplit_Combine_AnyTwo(t *testing.T) {
  secret := []byte("supersecretdata")
  shares := Split(secret, 3, 2)
  
  tests := []struct {
    name   string
    indices []int  // which shares to use
  }{
    {"shares_0_1", []int{0, 1}},
    {"shares_0_2", []int{0, 2}},
    {"shares_1_2", []int{1, 2}},
  }
  
  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      selected := make([][]byte, 0, len(tt.indices))
      for _, i := range tt.indices {
        selected = append(selected, shares[i])
      }
      
      recovered, err := Combine(selected)
      if err != nil {
        t.Fatalf("Combine failed: %v", err)
      }
      
      if !bytes.Equal(recovered, secret) {
        t.Errorf("recovered secret doesn't match original")
      }
    })
  }
}

func TestSplit_OnlyOneShare_Fails(t *testing.T) {
  secret := []byte("supersecretdata")
  shares := Split(secret, 3, 2)
  
  // Attempt to recover with only 1 share (threshold=2)
  recovered, err := Combine([][]byte{shares[0]})
  
  if err == nil {
    t.Fatal("expected error when combining only 1 share with threshold=2")
  }
  
  if len(recovered) > 0 {
    t.Errorf("should not recover secret with insufficient shares")
  }
}

func TestSplit_AllThreeShares(t *testing.T) {
  secret := []byte("supersecretdata")
  shares := Split(secret, 3, 2)
  
  // Should work with all 3 shares too
  recovered, err := Combine(shares)
  if err != nil {
    t.Fatalf("Combine with all shares failed: %v", err)
  }
  
  if !bytes.Equal(recovered, secret) {
    t.Errorf("recovered secret doesn't match original")
  }
}
```

**TOTP Tests** (`internal/services/twofaService/totp/totp_test.go`):

```go
package totp

import (
  "testing"
  "time"
)

func TestGenerateSecret(t *testing.T) {
  secret := GenerateSecret()
  
  if len(secret) != 20 {
    t.Errorf("expected 20-byte secret, got %d", len(secret))
  }
  
  // Ensure it's not empty/all zeros
  zero := make([]byte, 20)
  if bytes.Equal(secret, zero) {
    t.Errorf("secret is all zeros")
  }
}

func TestGenerateAndVerify(t *testing.T) {
  secret := GenerateSecret()
  now := time.Now()
  
  // Generate code at current time
  code := GenerateCode(secret, now)
  
  // Should verify at same time
  if !Verify(secret, code, now) {
    t.Errorf("failed to verify code at generation time")
  }
  
  // Should verify within ±1 window (±30 sec)
  if !Verify(secret, code, now.Add(-30*time.Second)) {
    t.Errorf("failed to verify at -30 sec window")
  }
  
  if !Verify(secret, code, now.Add(30*time.Second)) {
    t.Errorf("failed to verify at +30 sec window")
  }
  
  // Should NOT verify outside window (±2 periods)
  if Verify(secret, code, now.Add(-65*time.Second)) {
    t.Errorf("incorrectly verified code at -65 sec")
  }
}

func TestVerify_WrongCode(t *testing.T) {
  secret := GenerateSecret()
  now := time.Now()
  
  wrongCode := "000000"
  
  if Verify(secret, wrongCode, now) {
    t.Errorf("incorrectly verified wrong code")
  }
}

func TestProvisioningURI(t *testing.T) {
  secret := GenerateSecret()
  issuer := "MPC-2FA"
  account := "user@example.com"
  
  uri := ProvisioningURI(secret, issuer, account)
  
  if !bytes.HasPrefix(uri, []byte("otpauth://totp/")) {
    t.Errorf("invalid provisioning URI format")
  }
  
  // URI should contain escaped account and issuer
  if !bytes.Contains(uri, []byte(issuer)) {
    t.Errorf("provisioning URI missing issuer")
  }
}
```

**AES Encryption Tests** (`internal/crypto/aes_test.go`):

```go
package crypto

import (
  "bytes"
  "testing"
)

func TestEncryptDecrypt(t *testing.T) {
  key := make([]byte, 32) // AES-256
  for i := range key {
    key[i] = byte(i)
  }
  
  plaintext := []byte("secret data to encrypt")
  
  ciphertext, nonce, err := Encrypt(plaintext, key)
  if err != nil {
    t.Fatalf("Encrypt failed: %v", err)
  }
  
  // Nonce should be unique and non-empty
  if len(nonce) != 12 {
    t.Errorf("expected 12-byte nonce, got %d", len(nonce))
  }
  
  // Decrypt
  decrypted, err := Decrypt(ciphertext, nonce, key)
  if err != nil {
    t.Fatalf("Decrypt failed: %v", err)
  }
  
  if !bytes.Equal(decrypted, plaintext) {
    t.Errorf("decrypted data doesn't match original")
  }
}

func TestDecrypt_WrongKey_Fails(t *testing.T) {
  key := make([]byte, 32)
  plaintext := []byte("secret data")
  
  ciphertext, nonce, _ := Encrypt(plaintext, key)
  
  // Try with different key
  wrongKey := make([]byte, 32)
  for i := range wrongKey {
    wrongKey[i] = 0xFF
  }
  
  _, err := Decrypt(ciphertext, nonce, wrongKey)
  if err == nil {
    t.Fatal("expected error when decrypting with wrong key")
  }
}

func TestEncrypt_UniquNonce(t *testing.T) {
  key := make([]byte, 32)
  plaintext := []byte("data")
  
  nonce1 := make([]byte, 0)
  nonce2 := make([]byte, 0)
  
  _, nonce1, _ = Encrypt(plaintext, key)
  _, nonce2, _ = Encrypt(plaintext, key)
  
  if bytes.Equal(nonce1, nonce2) {
    t.Errorf("nonces should be unique for each encryption")
  }
}
```

## Password Validation Tests

**Complete example** (`internal/services/authService/password_validation_test.go`):

```go
package authService

import "testing"

func TestValidatePassword_AllRules(t *testing.T) {
  tests := []struct {
    name      string
    password  string
    wantError bool
  }{
    // Valid cases
    {"all_requirements_met", "MySecure@Pwd123", false},
    {"special_char_at_end", "ValidPass@123!", false},
    {"all_special_chars", "P@ssw0rd!#$", false},
    
    // Length violations
    {"too_short_11_chars", "Secur@Pass12", true},
    {"empty_password", "", true},
    
    // Missing uppercase
    {"no_uppercase", "mysecure@pass123", true},
    
    // Missing lowercase
    {"no_lowercase", "MYSECURE@PASS123", true},
    
    // Missing digit
    {"no_digit", "MySecurePass@", true},
    
    // Missing special character
    {"no_special_char", "MySecurePass123", true},
    
    // Sequence violations (4+ chars)
    {"numeric_sequence_1234", "Seq@1234pass", true},
    {"numeric_sequence_5678", "Seq@5678pass", true},
    {"alphabetic_sequence_abcd", "Seq@abcdpass", true},
    {"alphabetic_sequence_xyz", "Seq@xyzpass", true},
    {"keyboard_qwer", "Pass@qwerty123", true},
    {"keyboard_asdf", "Pass@asdfgh123", true},
    
    // Sequence accepted (3 chars)
    {"numeric_3chars_123", "Pass@123456", false},
    {"alphabetic_3chars_abc", "Pass@abcdef", false},
    
    // Reverse sequences
    {"reverse_numeric_4321", "Seq@4321pass", true},
    {"reverse_alphabetic_dcba", "Seq@dcbapass", true},
  }
  
  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      err := ValidatePassword(tt.password)
      if (err != nil) != tt.wantError {
        t.Errorf("ValidatePassword(%q) error = %v, wantError %v", tt.password, err, tt.wantError)
      }
    })
  }
}
```

## Error Handling Tests

**Pattern for testing error cases**:
```go
func TestRegister_InvalidPassword(t *testing.T) {
  svc := &AuthService{storage: &mockStorage{}}
  
  // Test each password validation failure
  tests := []struct {
    password string
    expectedErr error
  }{
    {"short", ErrInvalidPassword},
    {"NoSpecial123", ErrInvalidPassword},
    {"Pass@123seq1234", ErrInvalidPassword}, // contains 1234 sequence
  }
  
  for _, tt := range tests {
    _, err := svc.Register(context.Background(), "user@example.com", tt.password)
    if !errors.Is(err, tt.expectedErr) {
      t.Errorf("expected %v, got %v", tt.expectedErr, err)
    }
  }
}
```

## Coverage

**Requirements:** No strict coverage target enforced, but aim for:
- 100% coverage of crypto functions (Shamir, TOTP, AES, password validation)
- 80%+ coverage of service business logic
- 70%+ coverage overall for critical paths

**View Coverage:**
```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View in terminal
go tool cover -func=coverage.out

# View in browser (HTML)
go tool cover -html=coverage.out -o coverage.html
open coverage.html

# Coverage by package
go test -cover ./internal/services/authService/...
```

## What NOT to Test

- **Standard library functions** — don't test `bcrypt.GenerateFromPassword` behavior
- **Database driver functions** — don't test `pgx.QueryRow` behavior directly (integration tests only)
- **gRPC framework code** — don't test how `grpc.NewServer` works
- **External service APIs** — use mocks/stubs instead

---

*Testing analysis: 2026-04-11*
