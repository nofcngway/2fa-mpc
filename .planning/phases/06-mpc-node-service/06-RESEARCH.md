# Phase 6: MPC Node Service - Research

**Researched:** 2026-04-12
**Domain:** AES-256-GCM encryption, PostgreSQL CRUD, gRPC interceptors, Go crypto/aes + crypto/cipher
**Confidence:** HIGH

## Summary

Phase 6 implements the MPC Node service business logic on top of existing Phase 1 scaffolding. All structural code exists: proto definitions, generated gRPC stubs (returning Unimplemented), domain models, config loading, bootstrap wiring, and PostgreSQL table creation with the UNIQUE(user_id, share_index) constraint. The work is purely filling in the implementation: (1) AES-256-GCM encrypt/decrypt helpers in the service layer, (2) three PostgreSQL CRUD methods in storage, (3) three gRPC handler implementations delegating to the service, (4) a shared-secret auth interceptor in the middleware package, and (5) comprehensive tests.

The Go standard library provides everything needed for encryption (`crypto/aes`, `crypto/cipher`, `crypto/rand`). No external crypto dependencies are required. The project already uses `gotest.tools/v3` for assertions and `minimock/v3` for mock generation -- both must be added to `mpc/go.mod` as test dependencies. The existing code patterns from auth and twofa services provide clear templates for test structure, mock usage, and interceptor design.

**Primary recommendation:** Implement in two plans -- Plan 1 covers storage CRUD + service layer (encrypt/decrypt + business methods) + tests; Plan 2 covers auth interceptor + gRPC handlers + handler tests + bootstrap wiring updates.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Encryption/decryption lives in the service layer (MPCService). Storage layer is pure CRUD -- receives and returns already-encrypted bytes + nonce.
- **D-02:** Encryption key validated at startup -- bootstrap checks key length = 32 bytes when creating MPCService. Service refuses to start with invalid key size.
- **D-03:** Corrupted ciphertext during RetrieveShare returns gRPC `codes.Internal`. Log the decryption failure context (without share data).
- **D-04:** Shared secret interceptor protects ALL gRPC methods (StoreShare, RetrieveShare, DeleteShare). Health check excluded automatically.
- **D-05:** Interceptor receives expected shared secret from config at creation time. Uses `subtle.ConstantTimeCompare`. Reads client secret from gRPC metadata "authorization" header.
- **D-06:** DeleteShare operates by user_id only -- `DELETE FROM shares WHERE user_id = $1`.
- **D-07:** RetrieveShare returns gRPC `codes.NotFound` when no share exists for given user_id + share_index.
- **D-08:** DeleteShare returns silent success when user has no shares to delete (idempotent).
- **D-09:** Storage tested via interface mocks in service tests. Define Storage interface with CreateShare, GetShare, DeleteSharesByUserID methods.
- **D-10:** Comprehensive test coverage (25+ tests) for defense.

### Claude's Discretion
- Exact encrypt/decrypt helper function signatures within MPCService
- Error message wording (must not leak internal state per SEC-02)
- Whether to use `google.uuid` for share ID generation or let PostgreSQL generate
- Kafka audit event publishing structure (fire-and-forget pattern)
- Prometheus metric label design

### Deferred Ideas (OUT OF SCOPE)
- TwoFA to MPC integration (Phase 7)
- mTLS between services (v2 requirement ASEC-01)
- Prometheus metrics (Phase 9)
- Kafka audit events (Phase 9)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| MPC-01 | StoreShare -- encrypt share data with AES-256-GCM (unique nonce via crypto/rand), store encrypted_data + nonce in PostgreSQL | Go stdlib crypto/aes + crypto/cipher, 12-byte nonce from crypto/rand, storage CreateShare with pgx |
| MPC-02 | RetrieveShare -- read encrypted_data + nonce, decrypt, return share data | Storage GetShare by user_id + share_index, cipher.AEAD.Open for decryption |
| MPC-03 | DeleteShare -- delete all shares for a user from this node | Storage DeleteSharesByUserID, DELETE FROM shares WHERE user_id = $1 |
| MPC-04 | Unique constraint on (user_id, share_index) per node | Already exists in initTables (Phase 1 scaffolding), detect pgx duplicate key error |
| MPC-05 | gRPC interceptor validates shared secret via metadata ("authorization" header) | grpc.metadata.FromIncomingContext, subtle.ConstantTimeCompare, grpc.ChainUnaryInterceptor |
| MPC-06 | AES-256-GCM encryption key loaded from config (ENCRYPTION_KEY), nonce never reused | Config already loads Node.EncryptionKey, bootstrap validates 32-byte length, crypto/rand for unique nonce |
</phase_requirements>

## Standard Stack

### Core (Already in go.mod)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| crypto/aes | stdlib | AES block cipher | Go standard library, no external dependency needed [VERIFIED: Go stdlib] |
| crypto/cipher | stdlib | GCM authenticated encryption mode | Go standard library, provides cipher.NewGCM [VERIFIED: Go stdlib] |
| crypto/rand | stdlib | Cryptographically secure random nonce generation | Go standard library [VERIFIED: Go stdlib] |
| crypto/subtle | stdlib | Constant-time comparison for shared secret | Go standard library, prevents timing attacks [VERIFIED: Go stdlib] |
| github.com/jackc/pgx/v5 | v5.9.1 | PostgreSQL driver with connection pooling | Already in go.mod, project standard [VERIFIED: mpc/go.mod] |
| github.com/google/uuid | -- | UUID generation for share IDs | Already used in auth service, needs adding to mpc/go.mod [VERIFIED: auth/go.mod] |
| google.golang.org/grpc | v1.80.0 | gRPC framework, metadata, interceptors | Already in go.mod [VERIFIED: mpc/go.mod] |

### Test Dependencies (Need adding to mpc/go.mod)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| gotest.tools/v3 | v3.5.2 | Assert helpers (assert.NilError, assert.Equal, assert.Assert) | All test files [VERIFIED: auth/go.mod] |
| github.com/gojuno/minimock/v3 | v3.4.7 | Mock generation for Storage interface | Service unit tests [VERIFIED: auth/go.mod, minimock CLI v3.4.7 installed] |

### Installation
```bash
cd mpc && go get gotest.tools/v3@v3.5.2 github.com/gojuno/minimock/v3@v3.4.7 github.com/google/uuid
```

## Architecture Patterns

### Existing Project Structure (Phase 1 scaffolding -- files to modify)
```
mpc/
├── internal/
│   ├── api/mpc_service_api/
│   │   ├── mpc_service_api.go     # MPCServiceAPI struct (EXISTS)
│   │   ├── store_share.go         # Handler stub → implement
│   │   ├── retrieve_share.go      # Handler stub → implement
│   │   └── delete_share.go        # Handler stub → implement
│   ├── bootstrap/bootstrap.go     # DI wiring → add key validation + chain interceptors
│   ├── middleware/
│   │   ├── interceptors.go        # LoggingInterceptor (EXISTS) → add AuthInterceptor
│   │   └── interceptors_test.go   # NEW: interceptor tests
│   ├── models/models.go           # Share model (EXISTS, complete)
│   ├── services/mpcService/
│   │   ├── mpc_service.go         # Service struct (EXISTS) → add encrypt/decrypt + methods
│   │   ├── store_share.go         # NEW: StoreShare business logic
│   │   ├── retrieve_share.go      # NEW: RetrieveShare business logic
│   │   ├── delete_share.go        # NEW: DeleteShare business logic
│   │   ├── encrypt.go             # NEW: AES-256-GCM encrypt/decrypt helpers
│   │   ├── encrypt_test.go        # NEW: encryption roundtrip tests
│   │   ├── store_share_test.go    # NEW: service tests with mocked storage
│   │   ├── retrieve_share_test.go # NEW: service tests
│   │   ├── delete_share_test.go   # NEW: service tests
│   │   └── mocks/                 # NEW: minimock-generated Storage mock
│   ├── storage/pgstorage/
│   │   ├── pgstorage.go           # PGStorage + initTables (EXISTS)
│   │   ├── share.go               # NEW: CreateShare, GetShare, DeleteSharesByUserID
│   │   └── share_test.go          # OPTIONAL: integration tests (need live DB)
│   └── pb/                        # Generated protobuf code (EXISTS, complete)
├── config/config.go               # Config loading (EXISTS, complete)
└── go.mod                         # Needs test deps added
```

### Pattern 1: Service-Layer Encryption (D-01)
**What:** Service encrypts plaintext share data before passing to storage. Storage is pure CRUD on encrypted bytes.
**When to use:** Always -- this is a locked decision.
**Example:**
```go
// Source: Go stdlib crypto/aes + crypto/cipher documentation [VERIFIED: Go stdlib docs]
func (s *MPCService) encrypt(plaintext []byte) (ciphertext, nonce []byte, err error) {
    block, err := aes.NewCipher(s.encryptionKey)
    if err != nil {
        return nil, nil, fmt.Errorf("create cipher: %w", err)
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, nil, fmt.Errorf("create GCM: %w", err)
    }

    nonce = make([]byte, gcm.NonceSize()) // 12 bytes for GCM
    if _, err := rand.Read(nonce); err != nil {
        return nil, nil, fmt.Errorf("generate nonce: %w", err)
    }

    ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
    return ciphertext, nonce, nil
}

func (s *MPCService) decrypt(ciphertext, nonce []byte) ([]byte, error) {
    block, err := aes.NewCipher(s.encryptionKey)
    if err != nil {
        return nil, fmt.Errorf("create cipher: %w", err)
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, fmt.Errorf("create GCM: %w", err)
    }

    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return nil, fmt.Errorf("decrypt: %w", err)
    }

    return plaintext, nil
}
```

### Pattern 2: Storage Interface + Mocks (D-09)
**What:** Define Storage interface in service package, use minimock to generate mocks for unit testing.
**When to use:** Service tests -- mock storage to isolate business logic.
**Example:**
```go
// In mpc_service.go
type Storage interface {
    CreateShare(ctx context.Context, share *models.Share) error
    GetShare(ctx context.Context, userID string, shareIndex int) (*models.Share, error)
    DeleteSharesByUserID(ctx context.Context, userID string) (int64, error)
}

// Generate mock:
//go:generate minimock -i Storage -o ./mocks -g -s _mock.go
```

### Pattern 3: Auth Interceptor with Shared Secret (D-04, D-05)
**What:** gRPC unary interceptor that validates shared secret from metadata before allowing request processing.
**When to use:** All MPC gRPC methods except health check.
**Example:**
```go
// Source: google.golang.org/grpc/metadata docs [VERIFIED: grpc Go docs]
func AuthInterceptor(expectedSecret string) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        md, ok := metadata.FromIncomingContext(ctx)
        if !ok {
            return nil, status.Error(codes.Unauthenticated, "missing metadata")
        }

        values := md.Get("authorization")
        if len(values) == 0 {
            return nil, status.Error(codes.Unauthenticated, "missing authorization")
        }

        if subtle.ConstantTimeCompare([]byte(values[0]), []byte(expectedSecret)) != 1 {
            return nil, status.Error(codes.Unauthenticated, "invalid authorization")
        }

        return handler(ctx, req)
    }
}
```

### Pattern 4: Chaining Interceptors (Bootstrap Update)
**What:** Use `grpc.ChainUnaryInterceptor` to combine auth + logging interceptors.
**Example:**
```go
// Source: google.golang.org/grpc ChainUnaryInterceptor [VERIFIED: grpc Go docs]
server := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        middleware.AuthInterceptor(cfg.SharedSecret),
        middleware.LoggingInterceptor,
    ),
)
```

### Pattern 5: Detecting Duplicate Key in pgx (MPC-04)
**What:** Detect PostgreSQL unique constraint violation when storing duplicate (user_id, share_index).
**Example:**
```go
// Source: pgx error handling [VERIFIED: pgx v5 docs]
import "github.com/jackc/pgx/v5/pgconn"

func (ps *PGStorage) CreateShare(ctx context.Context, share *models.Share) error {
    _, err := ps.pool.Exec(ctx,
        `INSERT INTO shares (id, user_id, share_index, encrypted_data, nonce, created_at)
         VALUES ($1, $2, $3, $4, $5, $6)`,
        share.ID, share.UserID, share.ShareIndex, share.EncryptedData, share.Nonce, share.CreatedAt,
    )
    if err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" {
            return ErrDuplicateShare
        }
        return err
    }
    return nil
}
```

### Anti-Patterns to Avoid
- **Encrypting in storage layer:** Violates D-01. Storage receives pre-encrypted bytes only.
- **Reusing nonce:** Each StoreShare call MUST generate a fresh 12-byte nonce. Never derive from user_id or share_index.
- **Logging share data or encryption key:** Violates SEC-05, CLAUDE.md explicit prohibition.
- **String comparison for shared secret:** Must use `subtle.ConstantTimeCompare` to prevent timing attacks (D-05).
- **Using `grpc.UnaryInterceptor` with two interceptors:** Only one is allowed; use `grpc.ChainUnaryInterceptor` instead.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| AES-256-GCM | Custom encryption | `crypto/aes` + `cipher.NewGCM` (Go stdlib) | Audited implementation, GCM provides authentication [VERIFIED: Go stdlib] |
| Nonce generation | Sequential or derived nonces | `crypto/rand.Read` | Cryptographically secure, no reuse risk [VERIFIED: Go stdlib] |
| UUID generation | Custom ID format | `github.com/google/uuid` | RFC 4122 compliant, project standard [VERIFIED: used in auth service] |
| Mock generation | Manual mocks | `minimock` CLI + `//go:generate` | Project standard, type-safe mocks [VERIFIED: auth service pattern] |
| Timing-safe comparison | `==` operator | `crypto/subtle.ConstantTimeCompare` | Prevents timing side-channel attacks [VERIFIED: Go stdlib] |
| Interceptor chaining | Custom middleware chain | `grpc.ChainUnaryInterceptor` | Built into gRPC-Go, handles ordering correctly [VERIFIED: grpc v1.80.0] |

## Common Pitfalls

### Pitfall 1: GCM Nonce Size Hardcoding
**What goes wrong:** Hardcoding nonce size as 12 instead of using `gcm.NonceSize()`.
**Why it happens:** GCM standard nonce is 12 bytes, developers skip the method call.
**How to avoid:** Always use `gcm.NonceSize()` to get the nonce size from the cipher instance.
**Warning signs:** Magic number 12 appearing in nonce allocation.

### Pitfall 2: pgx Unique Constraint Error Detection
**What goes wrong:** Using string matching on error messages instead of PostgreSQL error codes.
**Why it happens:** Not knowing pgx exposes `pgconn.PgError` with structured error codes.
**How to avoid:** Use `errors.As(err, &pgErr)` and check `pgErr.Code == "23505"` (unique_violation).
**Warning signs:** `strings.Contains(err.Error(), "unique")` or similar fragile patterns.

### Pitfall 3: Single Interceptor Limitation
**What goes wrong:** Passing both auth and logging interceptors via separate `grpc.UnaryInterceptor` options -- only the last one takes effect.
**Why it happens:** `grpc.UnaryInterceptor` accepts exactly one interceptor. Multiple calls overwrite.
**How to avoid:** Use `grpc.ChainUnaryInterceptor(auth, logging)` which chains them in order.
**Warning signs:** One interceptor silently not executing.

### Pitfall 4: Encryption Key as String vs Bytes
**What goes wrong:** Config loads encryption key as string, but AES needs exactly 32 bytes. UTF-8 multi-byte characters could cause length mismatch.
**Why it happens:** YAML config stores string, `[]byte(cfg.Node.EncryptionKey)` conversion.
**How to avoid:** Validate `len([]byte(key)) == 32` at bootstrap (D-02). Consider hex-encoded key in config and decode at load time.
**Warning signs:** Service panics at runtime with "invalid key size" from `aes.NewCipher`.

### Pitfall 5: Missing pgconn Import for Error Handling
**What goes wrong:** pgconn package not imported, cannot type-assert PostgreSQL errors.
**Why it happens:** pgconn is a sub-package of pgx, not automatically imported.
**How to avoid:** Add `github.com/jackc/pgx/v5/pgconn` to imports in storage layer.
**Warning signs:** Compile error on `pgconn.PgError`.

### Pitfall 6: Health Check Blocked by Auth Interceptor
**What goes wrong:** Auth interceptor rejects health check probes that don't carry shared secret.
**Why it happens:** `grpc.ChainUnaryInterceptor` applies to ALL registered services including health.
**How to avoid:** In the auth interceptor, check `info.FullMethod` -- skip auth for `/grpc.health.v1.Health/Check`. Per D-04, health check is excluded because it's a separate gRPC service, but the interceptor applies server-wide.
**Warning signs:** Health check returns Unauthenticated.

## Code Examples

### AES-256-GCM Encrypt/Decrypt
```go
// Source: Go stdlib crypto/aes, crypto/cipher, crypto/rand [VERIFIED: Go stdlib]
import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "fmt"
)

func (s *MPCService) encrypt(plaintext []byte) (ciphertext, nonce []byte, err error) {
    block, err := aes.NewCipher(s.encryptionKey)
    if err != nil {
        return nil, nil, fmt.Errorf("create cipher: %w", err)
    }
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, nil, fmt.Errorf("create GCM: %w", err)
    }
    nonce = make([]byte, gcm.NonceSize())
    if _, err := rand.Read(nonce); err != nil {
        return nil, nil, fmt.Errorf("generate nonce: %w", err)
    }
    ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
    return ciphertext, nonce, nil
}

func (s *MPCService) decrypt(ciphertext, nonce []byte) ([]byte, error) {
    block, err := aes.NewCipher(s.encryptionKey)
    if err != nil {
        return nil, fmt.Errorf("create cipher: %w", err)
    }
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, fmt.Errorf("create GCM: %w", err)
    }
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return nil, fmt.Errorf("decrypt: %w", err)
    }
    return plaintext, nil
}
```

### Storage CRUD Methods
```go
// Source: pgx v5 patterns [VERIFIED: existing auth/twofa storage code in project]
func (ps *PGStorage) CreateShare(ctx context.Context, share *models.Share) error {
    _, err := ps.pool.Exec(ctx,
        `INSERT INTO shares (id, user_id, share_index, encrypted_data, nonce, created_at)
         VALUES ($1, $2, $3, $4, $5, $6)`,
        share.ID, share.UserID, share.ShareIndex,
        share.EncryptedData, share.Nonce, share.CreatedAt,
    )
    return err
}

func (ps *PGStorage) GetShare(ctx context.Context, userID string, shareIndex int) (*models.Share, error) {
    row := ps.pool.QueryRow(ctx,
        `SELECT id, user_id, share_index, encrypted_data, nonce, created_at
         FROM shares WHERE user_id = $1 AND share_index = $2`,
        userID, shareIndex,
    )
    var s models.Share
    err := row.Scan(&s.ID, &s.UserID, &s.ShareIndex, &s.EncryptedData, &s.Nonce, &s.CreatedAt)
    if err != nil {
        return nil, err
    }
    return &s, nil
}

func (ps *PGStorage) DeleteSharesByUserID(ctx context.Context, userID string) (int64, error) {
    tag, err := ps.pool.Exec(ctx,
        `DELETE FROM shares WHERE user_id = $1`, userID,
    )
    if err != nil {
        return 0, err
    }
    return tag.RowsAffected(), nil
}
```

### Test Pattern (matching project conventions)
```go
// Source: auth/internal/services/authService/*_test.go patterns [VERIFIED: codebase]
package mpcService_test

import (
    "context"
    "testing"

    "github.com/gojuno/minimock/v3"
    "gotest.tools/v3/assert"

    "github.com/vbncursed/vkr/mpc/internal/models"
    "github.com/vbncursed/vkr/mpc/internal/services/mpcService"
    "github.com/vbncursed/vkr/mpc/internal/services/mpcService/mocks"
)

type storeSuite struct {
    mc      *minimock.Controller
    storage *mocks.StorageMock
    service *mpcService.MPCService
}

func newStoreSuite(t *testing.T) *storeSuite {
    t.Helper()
    mc := minimock.NewController(t)
    storage := mocks.NewStorageMock(mc)
    key := make([]byte, 32) // test key
    copy(key, "test-encryption-key-32-bytes!!")
    service := mpcService.NewMPCService(storage, key, 1)
    return &storeSuite{mc: mc, storage: storage, service: service}
}
```

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + gotest.tools/v3 assertions + minimock/v3 mocks |
| Config file | None needed -- Go testing is built-in |
| Quick run command | `cd mpc && go test ./internal/services/mpcService/... ./internal/middleware/... -count=1` |
| Full suite command | `cd mpc && go test ./... -count=1 -v` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| MPC-01 | StoreShare encrypts + stores | unit | `cd mpc && go test ./internal/services/mpcService/ -run TestStoreShare -count=1` | Wave 0 |
| MPC-01 | AES-256-GCM encrypt roundtrip | unit | `cd mpc && go test ./internal/services/mpcService/ -run TestEncrypt -count=1` | Wave 0 |
| MPC-02 | RetrieveShare decrypts + returns | unit | `cd mpc && go test ./internal/services/mpcService/ -run TestRetrieveShare -count=1` | Wave 0 |
| MPC-03 | DeleteShare removes by user_id | unit | `cd mpc && go test ./internal/services/mpcService/ -run TestDeleteShare -count=1` | Wave 0 |
| MPC-04 | Duplicate (user_id, share_index) rejected | unit | `cd mpc && go test ./internal/services/mpcService/ -run TestDuplicate -count=1` | Wave 0 |
| MPC-05 | Auth interceptor validates secret | unit | `cd mpc && go test ./internal/middleware/ -run TestAuth -count=1` | Wave 0 |
| MPC-06 | Key from config, unique nonce | unit | `cd mpc && go test ./internal/services/mpcService/ -run TestNonce -count=1` | Wave 0 |

### Sampling Rate
- **Per task commit:** `cd mpc && go test ./internal/services/mpcService/... ./internal/middleware/... -count=1`
- **Per wave merge:** `cd mpc && go test ./... -count=1 -v`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `mpc/internal/services/mpcService/mocks/` -- minimock-generated Storage mock (run `go generate`)
- [ ] Test deps in go.mod: `gotest.tools/v3`, `minimock/v3`, `google/uuid`

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | Shared-secret auth interceptor with constant-time comparison (crypto/subtle) |
| V3 Session Management | no | N/A -- MPC nodes are stateless per-request |
| V4 Access Control | yes | All gRPC methods protected by auth interceptor (D-04) |
| V5 Input Validation | yes | Validate user_id format (UUID), share_index range, share_data non-empty |
| V6 Cryptography | yes | AES-256-GCM (Go stdlib), 12-byte nonce via crypto/rand, 32-byte key validated at startup |

### Known Threat Patterns for MPC Node

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Unauthorized share access | Spoofing | Shared secret in gRPC metadata, constant-time comparison |
| Nonce reuse (breaks GCM) | Tampering | Fresh nonce from crypto/rand per operation, never derived |
| Timing attack on auth | Information Disclosure | crypto/subtle.ConstantTimeCompare |
| Share data in logs | Information Disclosure | Never log share_data, encrypted_data, or encryption_key (SEC-05) |
| SQL injection | Tampering | Parameterized queries via pgx ($1, $2, ...) |
| Key exposure via error messages | Information Disclosure | Generic gRPC error messages, no internal state leaked (SEC-02) |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `pgconn.PgError.Code == "23505"` is the correct PostgreSQL error code for unique violation | Architecture Patterns | Duplicate detection would silently fail -- LOW risk, well-documented PostgreSQL code |
| A2 | `grpc.ChainUnaryInterceptor` is available in grpc v1.80.0 | Architecture Patterns | Would need manual chaining -- LOW risk, feature available since grpc-go v1.28 |
| A3 | Health check service is not affected by `ChainUnaryInterceptor` by default -- interceptor must explicitly skip health check methods | Common Pitfalls | Health check would fail if not excluded -- MEDIUM risk, documented in Pitfall 6 |

## Open Questions

1. **Encryption key encoding in config.yaml**
   - What we know: Config loads `encryption_key` as a Go string, converted to `[]byte`.
   - What's unclear: Should the key be stored as raw 32-char ASCII, hex-encoded (64 chars), or base64-encoded?
   - Recommendation: Use raw ASCII for simplicity (32 printable ASCII chars = 32 bytes). Validate length at bootstrap. This matches the existing `[]byte(cfg.Node.EncryptionKey)` pattern in bootstrap.

2. **Share ID generation: UUID in Go vs PostgreSQL DEFAULT**
   - What we know: Table has `id UUID PRIMARY KEY`, model has `ID string`.
   - What's unclear: Whether to generate UUID in Go code or use PostgreSQL's `gen_random_uuid()`.
   - Recommendation: Generate in Go with `uuid.New().String()` -- keeps storage layer pure SQL, matches auth service pattern, allows ID to be known before INSERT.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | All code | Yes | 1.26.2 | -- |
| minimock CLI | Mock generation | Yes | v3.4.7 | -- |
| PostgreSQL | Storage tests (integration only) | Not checked | -- | Unit tests use mocks, no live DB needed |

## Sources

### Primary (HIGH confidence)
- Go stdlib documentation (crypto/aes, crypto/cipher, crypto/rand, crypto/subtle) -- encryption patterns
- Existing codebase: mpc/ scaffolding from Phase 1 -- all file structures, models, config, proto verified by reading actual files
- Existing codebase: auth/ test patterns -- test structure, minimock usage, gotest.tools assertions

### Secondary (MEDIUM confidence)
- gRPC-Go documentation -- ChainUnaryInterceptor, metadata.FromIncomingContext
- pgx v5 documentation -- pgconn.PgError for constraint violation detection

### Tertiary (LOW confidence)
- None -- all claims verified against codebase or Go stdlib

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all Go stdlib, existing project deps verified in go.mod files
- Architecture: HIGH -- existing scaffolding code read and analyzed, patterns copied from auth/twofa services
- Pitfalls: HIGH -- based on direct code inspection and known Go crypto/gRPC patterns

**Research date:** 2026-04-12
**Valid until:** 2026-05-12 (stable Go stdlib, locked project dependencies)
