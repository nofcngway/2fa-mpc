# Phase 7: TwoFA Setup Flow - Research

**Researched:** 2026-04-12
**Domain:** Go gRPC service orchestration, Shamir secret distribution, backup code generation
**Confidence:** HIGH

## Summary

Phase 7 orchestrates 2FA setup: generate TOTP secret, Shamir split (2-of-3), distribute shares in parallel to 3 MPC nodes via gRPC, generate 10 backup codes (bcrypt-hashed), return provisioning URI. All crypto primitives (Shamir, TOTP) and MPC node service already exist from Phases 4-6. The existing TwoFA service scaffolding (Phase 1) provides empty interfaces, stub handlers, and bootstrap wiring that need to be filled in.

The primary work is orchestration logic: wiring MPC gRPC clients into the TwoFA service via bootstrap, implementing the Setup method with errgroup-based parallel share distribution and compensating rollback, adding backup code generation with bcrypt hashing, and ensuring secret zeroization. Proto changes are minimal (add `email` field to `Setup2FARequest`). Storage layer needs 3 methods implemented against already-existing tables.

**Primary recommendation:** Implement in 2 plans: (1) proto update + storage layer + MPC client wiring in bootstrap, (2) setup orchestration logic with parallel distribution, rollback, backup codes, zeroization, and comprehensive tests.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Parallel share distribution -- 3 goroutines via `errgroup` with shared context. All 3 StoreShare calls execute concurrently. If any fails, context is cancelled and remaining calls abort.
- **D-02:** Compensating delete on partial failure -- if any StoreShare fails, call DeleteShare(user_id) on ALL 3 nodes (idempotent per Phase 6 D-08). Ensures no orphaned shares remain in any node.
- **D-03:** gRPC clients to MPC nodes created at TwoFA service startup, configured via `config.yaml` `mpc_nodes` array (3 entries with `addr` and `shared_secret`). Shared secret sent in gRPC metadata "authorization" header per Phase 6 D-05.
- **D-04:** Timeout per MPC call -- use context with timeout from config (default 5s). Setup fails with `codes.Internal` if any node times out.
- **D-05:** Backup code format `xxxx-xxxx` -- 8 digits split by hyphen. Generated via `crypto/rand`, each half is 4 random digits (0000-9999). 10 codes per setup.
- **D-06:** Codes bcrypt-hashed (cost=12) before storage in `backup_codes` table. Plaintext codes returned to user in Setup2FAResponse only once -- never stored or logged.
- **D-07:** On comparison (Phase 8), strip hyphen before bcrypt check. Normalize input: remove hyphens, spaces, leading zeros preserved.
- **D-08:** `defer zeroize(secret)` immediately after `totp.GenerateSecret()` call. Zeroize function: loop over `[]byte`, set each to 0. Guarantees cleanup on success, error, and panic paths.
- **D-09:** Shares (`[]Share`) also zeroized after distribution -- `defer` for each share's `Data` field. Secret must not survive in any form after setup completes.
- **D-10:** Zeroize utility function in `twofa/internal/crypto/` package -- shared between setup (Phase 7) and verify (Phase 8).
- **D-11:** Add `email` field to `Setup2FARequest` proto message. Gateway/client passes email alongside user_id. No cross-service gRPC dependency on Auth.
- **D-12:** Duplicate setup prevention -- check `twofa_records` for existing record with `is_enabled=true`. If found, return `codes.AlreadyExists`. User must Disable2FA first, then re-setup.
- **D-13:** Storage interface methods: CreateTwoFARecord, GetTwoFARecord, StoreBatchBackupCodes.
- **D-14:** TwoFARecord initially created with `is_enabled=false`. Transitions to `true` only on first successful verification (Phase 8).
- **D-15:** Comprehensive tests (~20+) for defense.

### Claude's Discretion
- Exact errgroup pattern and error aggregation
- MPC gRPC client wrapper/pool design (connection management)
- Whether to use a separate `MPCClient` interface or call gRPC stubs directly
- Kafka audit event structure for `2fa.setup_started`, `2fa.setup_completed`, `2fa.setup_failed`
- Prometheus metric labels for setup operations
- Internal helper decomposition within Setup method

### Deferred Ideas (OUT OF SCOPE)
- OTP verification flow -- Phase 8
- Rate limiting -- Phase 8
- Disable 2FA -- Phase 8
- 2FA status check -- Phase 8
- Kafka audit events -- Phase 9
- Prometheus metrics -- Phase 9
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| 2FA-01 | User can setup 2FA -- TOTP secret generated, split via Shamir (2-of-3), shares sent to 3 MPC nodes, secret zeroized, provisioning URI returned | Shamir `Split()` and TOTP `GenerateSecret()`/`GenerateProvisioningURI()` exist. errgroup for parallel distribution. Zeroize utility needed. |
| 2FA-02 | Setup fails if any MPC node is unreachable (all 3 shares MUST be stored) | errgroup cancels on first error. Compensating DeleteShare on all nodes on any failure. |
| 2FA-08 | 10 backup codes generated on setup, each bcrypt-hashed, stored in PostgreSQL | `crypto/rand` for generation, `golang.org/x/crypto/bcrypt` for hashing (cost=12). `backup_codes` table exists. |
| SEC-04 | TOTP secret never persisted -- only transient in memory, zeroized after use | `defer zeroize(secret)` immediately after generation. Share Data also zeroized. |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

- **Clean Architecture**: handler -> service -> repository, dependencies via interfaces
- **DI through bootstrap**: each dependency created by factory in `internal/bootstrap/`
- **No ORM**: pgx directly, parameterized queries
- **No third-party Shamir**: custom implementation (already done)
- **TOTP secret NEVER persisted**: only transient in memory, zeroize after use
- **Logging**: slog structured, NEVER log secrets, passwords, shares, encryption keys
- **gRPC only**: no HTTP in services other than Gateway
- **Errors**: gRPC status codes (InvalidArgument, NotFound, Unauthenticated, AlreadyExists, Internal)
- **One file per gRPC method** in handler directory
- **One file per service method** in service directory
- **Test framework**: `gotest.tools/v3` with `minimock` for mock generation
- **bcrypt cost=12** for all password/code hashing

## Standard Stack

### Core (Already in Project)
| Library | Version | Purpose | Status |
|---------|---------|---------|--------|
| google.golang.org/grpc | v1.80.0 | gRPC framework, MPC client connections | In twofa/go.mod [VERIFIED: twofa/go.mod] |
| github.com/jackc/pgx/v5 | v5.9.1 | PostgreSQL driver for storage layer | In twofa/go.mod [VERIFIED: twofa/go.mod] |
| golang.org/x/sync | v0.20.0 | errgroup for parallel goroutine management | Indirect dep in twofa/go.mod [VERIFIED: twofa/go.mod] |
| gotest.tools/v3 | v3.5.2 | Test assertions | In twofa/go.mod [VERIFIED: twofa/go.mod] |
| google.golang.org/protobuf | v1.36.11 | Proto message definitions | In twofa/go.mod [VERIFIED: twofa/go.mod] |

### Needs Adding
| Library | Version | Purpose | Why |
|---------|---------|---------|-----|
| golang.org/x/crypto | latest | bcrypt for backup code hashing (cost=12) | NOT in twofa/go.mod. Required for D-06. [VERIFIED: grep twofa/go.mod] |
| github.com/google/uuid | latest | UUID generation for backup code IDs | NOT in twofa/go.mod. Needed for backup_codes.id. [VERIFIED: grep twofa/go.mod] |
| github.com/gojuno/minimock/v3 | latest | Mock generation for testing | Used in auth service tests, needed for twofa. [VERIFIED: auth register_test.go] |

**Note:** `golang.org/x/sync` is already an indirect dependency; it must be promoted to direct when importing `errgroup`. Run `go mod tidy` after adding the import.

**Installation:**
```bash
cd twofa
go get golang.org/x/crypto
go get github.com/google/uuid
go get github.com/gojuno/minimock/v3
```

## Architecture Patterns

### Existing Structure (from Phase 1 scaffolding)
```
twofa/
├── api/twofa_api/twofa_service.proto     # Needs email field in Setup2FARequest
├── api/models/models.proto                # TwoFARecord, BackupCode messages
├── internal/
│   ├── api/twofa_service_api/
│   │   ├── twofa_service_api.go          # Service interface (empty, fill in)
│   │   └── setup.go                      # Setup2FA handler (stub, implement)
│   ├── bootstrap/bootstrap.go            # DI wiring (needs MPC clients)
│   ├── crypto/
│   │   ├── shamir/shamir.go              # Split/Combine (Phase 4, complete)
│   │   └── totp/totp.go, uri.go          # GenerateSecret, GenerateProvisioningURI (Phase 5, complete)
│   ├── models/models.go                  # TwoFARecord, BackupCode domain models
│   ├── pb/twofa_api/                     # Generated protobuf code
│   ├── services/twofaService/
│   │   └── twofa_service.go              # Service struct + interfaces (empty, fill in)
│   └── storage/pgstorage/
│       └── pgstorage.go                  # PGStorage with tables (need methods)
```

### New Files to Create
```
twofa/internal/
├── crypto/zeroize.go                     # Zeroize utility (D-10)
├── services/twofaService/
│   ├── setup.go                          # Setup2FA business logic
│   ├── setup_test.go                     # Comprehensive tests (D-15)
│   ├── backup_codes.go                   # Backup code generation helper
│   └── mocks/                            # Generated mocks (minimock)
│       ├── storage_mock.go
│       ├── session_storage_mock.go
│       └── mpc_client_mock.go
└── storage/pgstorage/
    ├── twofa_record.go                   # CreateTwoFARecord, GetTwoFARecord
    └── backup_code.go                    # StoreBatchBackupCodes
```

### Pattern 1: errgroup for Parallel MPC Distribution
**What:** Use `golang.org/x/sync/errgroup` with shared context to distribute shares to 3 MPC nodes in parallel. On first failure, context cancellation aborts remaining calls.
**When to use:** D-01 mandates this pattern.
**Example:**
```go
// [VERIFIED: golang.org/x/sync already in go.sum]
import "golang.org/x/sync/errgroup"

g, ctx := errgroup.WithContext(ctx)

for i, share := range shares {
    i, share := i, share // capture loop vars
    g.Go(func() error {
        callCtx, cancel := context.WithTimeout(ctx, s.mpcTimeout)
        defer cancel()

        _, err := s.mpcClients[i].StoreShare(callCtx, &mpc_api.StoreShareRequest{
            UserId:     userID,
            ShareIndex: int32(share.Index),
            ShareData:  share.Data,
        })
        return err
    })
}

if err := g.Wait(); err != nil {
    // Compensating delete on ALL nodes (D-02)
    s.deleteSharesFromAllNodes(ctx, userID)
    return nil, fmt.Errorf("distribute shares: %w", err)
}
```

### Pattern 2: MPC Client Interface for Testability
**What:** Define an `MPCClient` interface wrapping the generated gRPC client. This enables mock injection for unit tests without real MPC nodes.
**When to use:** Claude's discretion area -- recommended for clean testing.
**Example:**
```go
// In twofa_service.go -- interface for MPC node gRPC client
//go:generate minimock -i MPCClient -o ./mocks/ -s _mock.go
type MPCClient interface {
    StoreShare(ctx context.Context, in *mpc_api.StoreShareRequest, opts ...grpc.CallOption) (*mpc_api.StoreShareResponse, error)
    RetrieveShare(ctx context.Context, in *mpc_api.RetrieveShareRequest, opts ...grpc.CallOption) (*mpc_api.RetrieveShareResponse, error)
    DeleteShare(ctx context.Context, in *mpc_api.DeleteShareRequest, opts ...grpc.CallOption) (*mpc_api.DeleteShareResponse, error)
}
```
This interface matches the generated `mpc_api.MPCNodeServiceClient` exactly, so real gRPC clients satisfy it without adapters. [VERIFIED: mpc_service_grpc.pb.go client interface]

### Pattern 3: gRPC Metadata for Shared Secret Auth
**What:** MPC nodes require shared secret in "authorization" metadata header (Phase 6 D-05). TwoFA service must attach this metadata to every outgoing MPC call.
**When to use:** Every gRPC call to MPC nodes.
**Example:**
```go
// Per-call metadata injection via grpc.PerRPCCredentials or manual:
import "google.golang.org/grpc/metadata"

func (s *TwoFAService) callWithAuth(ctx context.Context) context.Context {
    return metadata.AppendToOutgoingContext(ctx, "authorization", s.sharedSecret)
}
```
**Recommendation:** Use `grpc.WithPerRPCCredentials` at dial time so every call automatically includes the secret. Alternatively, use `grpc.WithUnaryInterceptor` on the client connection. This is cleaner than manual metadata per call. [ASSUMED]

### Pattern 4: Zeroize Utility
**What:** Zero-fill byte slices containing secrets to prevent memory leakage.
**When to use:** After TOTP secret generation (D-08) and after share distribution (D-09).
**Example:**
```go
// twofa/internal/crypto/zeroize.go
package crypto

// Zeroize overwrites all bytes in the slice with zeros.
func Zeroize(b []byte) {
    for i := range b {
        b[i] = 0
    }
}
```

### Pattern 5: Backup Code Generation
**What:** Generate 10 random codes in `xxxx-xxxx` format using `crypto/rand`.
**When to use:** During Setup2FA (D-05, D-06).
**Example:**
```go
import (
    "crypto/rand"
    "fmt"
    "math/big"

    "golang.org/x/crypto/bcrypt"
)

const BACKUP_CODE_COUNT = 10
const COST_BCRYPT = 12

func generateBackupCode() (string, error) {
    left, err := rand.Int(rand.Reader, big.NewInt(10000))
    if err != nil {
        return "", err
    }
    right, err := rand.Int(rand.Reader, big.NewInt(10000))
    if err != nil {
        return "", err
    }
    return fmt.Sprintf("%04d-%04d", left.Int64(), right.Int64()), nil
}
```

### Anti-Patterns to Avoid
- **Persisting TOTP secret anywhere:** Even temporarily in DB or Redis -- only in-process memory with defer zeroize [SEC-04]
- **Logging share data or secret bytes:** slog must never include these fields [CLAUDE.md]
- **Sequential MPC calls:** Must be parallel per D-01; sequential would add 3x latency
- **Ignoring partial failure cleanup:** If 2 of 3 StoreShare succeed but 1 fails, orphaned shares remain unless compensating delete runs [D-02]
- **Using math/rand for backup codes:** Must use `crypto/rand` for cryptographic randomness [D-05]

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Parallel goroutine management | Custom WaitGroup + error channels | `errgroup.WithContext` | Handles context cancellation, first-error propagation, goroutine lifecycle [VERIFIED: stdlib] |
| Password/code hashing | Custom hash | `golang.org/x/crypto/bcrypt` | Industry standard, constant-time comparison built in [VERIFIED: auth service pattern] |
| Random number generation | `math/rand` | `crypto/rand` | Cryptographic randomness required for backup codes [CITED: CLAUDE.md] |
| UUID generation | Custom ID generation | `github.com/google/uuid` | Collision-free, RFC 4122 compliant [VERIFIED: auth service pattern] |
| Mock generation | Hand-written mocks | `minimock` | Type-safe mocks, consistent with auth service pattern [VERIFIED: auth test pattern] |

## Common Pitfalls

### Pitfall 1: Forgotten Secret Zeroization on Error Path
**What goes wrong:** If setup fails between secret generation and share distribution, the secret may remain in memory.
**Why it happens:** Error return before reaching zeroize call.
**How to avoid:** `defer crypto.Zeroize(secret)` immediately after `totp.GenerateSecret()`, before any fallible operation (D-08).
**Warning signs:** Secret bytes not zeroed in test assertions.

### Pitfall 2: Loop Variable Capture in Goroutines
**What goes wrong:** All goroutines in the errgroup loop share the same `i` and `share` variable, leading to data races.
**Why it happens:** Go closure captures variable by reference, not value (pre-Go 1.22 behavior).
**How to avoid:** Shadow loop variables: `i, share := i, share` before `g.Go(func() ...)`. Note: Go 1.22+ has per-iteration scoping, but explicit capture is clearer. [VERIFIED: Go 1.26.2 has per-iteration scoping but explicit capture is the established pattern in this codebase]
**Warning signs:** Race detector failures in tests.

### Pitfall 3: Orphaned Shares on Partial Failure
**What goes wrong:** 2 of 3 nodes store shares, but node 3 fails. Without cleanup, partial shares persist.
**Why it happens:** No compensating transaction in distributed system.
**How to avoid:** On any StoreShare failure, call DeleteShare on ALL 3 nodes (idempotent). D-02 mandates this. [VERIFIED: Phase 6 DeleteShare is idempotent]

### Pitfall 4: Blocking Compensating Deletes on Failed Context
**What goes wrong:** The errgroup context is cancelled when a StoreShare fails. If you reuse this context for DeleteShare calls, they immediately fail.
**Why it happens:** errgroup.WithContext cancels the derived context on first error.
**How to avoid:** Use a fresh `context.Background()` with timeout for compensating delete calls, not the errgroup-derived context.
**Warning signs:** DeleteShare calls returning "context canceled" errors.

### Pitfall 5: Duplicate Backup Code IDs
**What goes wrong:** Batch insert of 10 backup codes with duplicate UUIDs fails.
**Why it happens:** Extremely unlikely with UUIDv4 but must handle.
**How to avoid:** Generate UUID per code in loop; pgx will report unique constraint violation if collision occurs.

### Pitfall 6: MPC Proto Import in TwoFA Module
**What goes wrong:** TwoFA service needs MPC proto-generated types (`StoreShareRequest`, etc.) but they are in a separate Go module (`github.com/vbncursed/vkr/mpc`).
**Why it happens:** Each service is a separate Go module.
**How to avoid:** Import the MPC module's generated pb package. Add `github.com/vbncursed/vkr/mpc` as a dependency in twofa/go.mod, or use a `replace` directive for local development. [VERIFIED: services are separate Go modules per CLAUDE.md]

## Code Examples

### MPC Client Connection at Bootstrap
```go
// [VERIFIED: mpc proto generates MPCNodeServiceClient with NewMPCNodeServiceClient(cc)]
import (
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    mpc_api "github.com/vbncursed/vkr/mpc/internal/pb/mpc_api"
)

func NewMPCClients(cfg *config.Config) ([]mpc_api.MPCNodeServiceClient, []io.Closer, error) {
    clients := make([]mpc_api.MPCNodeServiceClient, len(cfg.MPCNodes))
    conns := make([]io.Closer, len(cfg.MPCNodes))

    for i, node := range cfg.MPCNodes {
        conn, err := grpc.NewClient(node.Addr,
            grpc.WithTransportCredentials(insecure.NewCredentials()),
        )
        if err != nil {
            // Close already-opened connections
            for j := 0; j < i; j++ {
                conns[j].Close()
            }
            return nil, nil, fmt.Errorf("connect to MPC node %d: %w", i, err)
        }
        clients[i] = mpc_api.NewMPCNodeServiceClient(conn)
        conns[i] = conn
    }
    return clients, conns, nil
}
```

### Setup2FA Service Method Skeleton
```go
// twofa/internal/services/twofaService/setup.go
func (s *TwoFAService) Setup(ctx context.Context, userID, email string) (string, []string, error) {
    // 1. Check duplicate -- GetTwoFARecord, if is_enabled=true -> AlreadyExists
    // 2. Generate TOTP secret
    //    raw, base32, err := totp.GenerateSecret()
    //    defer crypto.Zeroize(raw)
    // 3. Split secret: shares, err := shamir.Split(raw, 3, 2)
    //    defer for each share: crypto.Zeroize(share.Data)
    // 4. Distribute shares in parallel via errgroup
    // 5. On failure: compensating delete on all nodes
    // 6. Create TwoFARecord (is_enabled=false)
    // 7. Generate 10 backup codes, bcrypt hash, batch store
    // 8. Build provisioning URI
    // 9. Return URI + plaintext backup codes
}
```

### Handler Error Mapping
```go
// twofa/internal/api/twofa_service_api/setup.go
// [VERIFIED: matches auth service handler pattern]
func (api *TwoFAServiceAPI) Setup2FA(ctx context.Context, req *pb.Setup2FARequest) (*pb.Setup2FAResponse, error) {
    if req.UserId == "" || req.Email == "" {
        return nil, status.Error(codes.InvalidArgument, "user_id and email are required")
    }

    uri, backupCodes, err := api.service.Setup(ctx, req.UserId, req.Email)
    if err != nil {
        if errors.Is(err, ErrAlreadyEnabled) {
            return nil, status.Error(codes.AlreadyExists, "2FA already enabled")
        }
        return nil, status.Error(codes.Internal, "internal error")
    }

    return &pb.Setup2FAResponse{
        ProvisioningUri: uri,
        BackupCodes:     backupCodes,
    }, nil
}
```

## State of the Art

| Aspect | Current State | Note |
|--------|---------------|------|
| errgroup | Standard Go concurrency pattern since Go 1.7+ | No changes needed [VERIFIED: golang.org/x/sync] |
| grpc.NewClient | Replaces deprecated grpc.Dial in recent grpc-go | Use `grpc.NewClient` not `grpc.Dial` [VERIFIED: grpc v1.80.0 in go.mod] |
| Go 1.22+ loop scoping | Per-iteration variable capture in for loops | Go 1.26.2 has this, but explicit capture is safe [VERIFIED: go version] |
| minimock v3 | Current mock generation tool in project | Established pattern in auth service [VERIFIED: auth tests] |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `grpc.WithPerRPCCredentials` is the cleanest approach for attaching shared secret to outgoing MPC calls | Architecture Patterns | LOW -- manual metadata.AppendToOutgoingContext works as fallback |
| A2 | MPC pb package can be imported cross-module via go.mod replace directive | Pitfall 6 | MEDIUM -- if module structure prevents import, need shared proto package |

## Open Questions

1. **Cross-module MPC proto import**
   - What we know: TwoFA and MPC are separate Go modules. TwoFA needs MPC's generated protobuf types.
   - What's unclear: Whether to use `replace` directive for local dev or publish/import the mpc module directly.
   - Recommendation: Use `replace` directive in twofa/go.mod pointing to `../mpc`. This is the standard pattern for multi-module monorepos.

2. **MPC timeout configuration**
   - What we know: D-04 says "timeout from config (default 5s)".
   - What's unclear: Whether to add a `mpc_timeout` field to config.yaml or hardcode as const.
   - Recommendation: Add `mpc_timeout: 5s` to config.yaml with `time.Duration` parsing, matching the pattern of other config values. Fall back to 5s const if not set.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | Build/test | Yes | 1.26.2 | -- |
| protoc | Proto regeneration | Yes | 34.1 | -- |
| minimock | Mock generation | Yes | (installed at ~/go/bin) | -- |
| PostgreSQL | Storage layer | Yes (via config) | -- | Docker compose |
| Redis | Session storage | Yes (via config) | -- | Optional (bootstrap handles nil) |

**Missing dependencies with no fallback:** None

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | gotest.tools/v3 + minimock v3 |
| Config file | None (standard `go test`) |
| Quick run command | `cd twofa && go test ./internal/services/twofaService/ -v -count=1` |
| Full suite command | `cd twofa && go test ./... -v -count=1` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| 2FA-01 | Setup happy path: secret generated, split, stored, URI + 10 codes returned | unit | `go test ./internal/services/twofaService/ -run TestSetup_Success -v` | Wave 0 |
| 2FA-02 | Any MPC node unreachable -> setup fails, compensating delete on all nodes | unit | `go test ./internal/services/twofaService/ -run TestSetup_PartialMPCFailure -v` | Wave 0 |
| 2FA-02 | All MPC nodes fail -> error returned, no shares persisted | unit | `go test ./internal/services/twofaService/ -run TestSetup_AllMPCFail -v` | Wave 0 |
| 2FA-08 | 10 backup codes generated, bcrypt-hashed, stored | unit | `go test ./internal/services/twofaService/ -run TestSetup_BackupCodes -v` | Wave 0 |
| 2FA-08 | Backup code format matches xxxx-xxxx | unit | `go test ./internal/services/twofaService/ -run TestBackupCode_Format -v` | Wave 0 |
| SEC-04 | Secret zeroized after setup | unit | `go test ./internal/services/twofaService/ -run TestSetup_Zeroization -v` | Wave 0 |
| 2FA-01 | Duplicate setup prevention (is_enabled=true -> AlreadyExists) | unit | `go test ./internal/services/twofaService/ -run TestSetup_DuplicateEnabled -v` | Wave 0 |

### Sampling Rate
- **Per task commit:** `cd twofa && go test ./internal/services/twofaService/ -v -count=1`
- **Per wave merge:** `cd twofa && go test ./... -v -count=1`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `twofa/internal/services/twofaService/setup_test.go` -- covers 2FA-01, 2FA-02, 2FA-08, SEC-04
- [ ] `twofa/internal/services/twofaService/mocks/` -- Storage, MPCClient, SessionStorage mocks
- [ ] `twofa/internal/crypto/zeroize_test.go` -- verify zeroize utility
- [ ] `golang.org/x/crypto` dependency added to go.mod

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | Not in scope (Auth service handles this) |
| V3 Session Management | No | Not in scope for setup flow |
| V4 Access Control | Yes | Duplicate setup prevention (D-12) |
| V5 Input Validation | Yes | user_id and email validation in handler |
| V6 Cryptography | Yes | Shamir split (custom GF(256)), bcrypt (cost=12), crypto/rand, zeroization |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Secret leakage via logs | Information Disclosure | slog NEVER logs secret/share bytes (CLAUDE.md) |
| Secret persistence in memory | Information Disclosure | defer zeroize(secret) + defer zeroize(share.Data) (D-08, D-09) |
| Partial share distribution | Denial of Service / Tampering | Compensating delete on all nodes (D-02) |
| Weak backup codes | Spoofing | crypto/rand (not math/rand), 10^8 keyspace per code |
| Replay of setup request | Tampering | Duplicate check via twofa_records (D-12) |
| SQL injection | Tampering | Parameterized queries via pgx ($1, $2) |

## Sources

### Primary (HIGH confidence)
- `twofa/go.mod` -- verified all existing dependencies and versions
- `twofa/internal/crypto/shamir/shamir.go` -- verified Split/Combine API signatures
- `twofa/internal/crypto/totp/totp.go` -- verified GenerateSecret returns ([]byte, string, error)
- `twofa/internal/crypto/totp/uri.go` -- verified GenerateProvisioningURI(secret, email) string
- `mpc/internal/pb/mpc_api/mpc_service_grpc.pb.go` -- verified MPCNodeServiceClient interface
- `mpc/api/mpc_api/mpc_service.proto` -- verified StoreShare/RetrieveShare/DeleteShare RPCs
- `twofa/api/twofa_api/twofa_service.proto` -- verified existing proto, Setup2FARequest lacks email
- `twofa/internal/storage/pgstorage/pgstorage.go` -- verified tables twofa_records and backup_codes exist
- `auth/internal/services/authService/` -- verified test patterns with minimock

### Secondary (MEDIUM confidence)
- `mpc/internal/middleware/interceptors.go` -- verified shared secret auth via "authorization" metadata

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all libraries verified in go.mod and codebase
- Architecture: HIGH -- patterns derived from existing auth service + Phase 1 scaffolding
- Pitfalls: HIGH -- derived from direct code reading and gRPC/errgroup semantics

**Research date:** 2026-04-12
**Valid until:** 2026-05-12 (stable Go ecosystem, no fast-moving dependencies)
