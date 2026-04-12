---
phase: 07-twofa-setup-flow
reviewed: 2026-04-12T12:00:00Z
depth: standard
files_reviewed: 15
files_reviewed_list:
  - twofa/api/mpc_api/mpc_service.proto
  - twofa/api/twofa_api/twofa_service.proto
  - twofa/cmd/app/main.go
  - twofa/config/config.go
  - twofa/internal/api/twofa_service_api/setup.go
  - twofa/internal/api/twofa_service_api/twofa_service_api.go
  - twofa/internal/bootstrap/bootstrap.go
  - twofa/internal/crypto/zeroize_test.go
  - twofa/internal/crypto/zeroize.go
  - twofa/internal/services/twofaService/backup_codes.go
  - twofa/internal/services/twofaService/setup_test.go
  - twofa/internal/services/twofaService/setup.go
  - twofa/internal/services/twofaService/twofa_service.go
  - twofa/internal/storage/pgstorage/backup_code.go
  - twofa/internal/storage/pgstorage/twofa_record.go
findings:
  critical: 2
  warning: 5
  info: 3
  total: 10
status: issues_found
---

# Phase 7: Code Review Report

**Reviewed:** 2026-04-12T12:00:00Z
**Depth:** standard
**Files Reviewed:** 15
**Status:** issues_found

## Summary

The TwoFA Setup Flow implementation is well-structured, following Clean Architecture with proper separation of concerns. The Shamir split, TOTP generation, backup code flow, and MPC node distribution are correctly orchestrated. Zeroization of secrets is consistently applied via defers. Test coverage is thorough with 13 test cases covering success, failure, partial failure, and compensating delete scenarios.

Key concerns: (1) the `base32Secret` string returned by `totp.GenerateSecret()` is never zeroized -- it persists in memory after Setup returns and is embedded in the provisioning URI, (2) the MPC node count is not validated, which could cause panics or silent data loss if misconfigured, and (3) the `os.Exit(1)` inside the goroutine in main.go will skip deferred cleanup functions.

## Critical Issues

### CR-01: base32Secret string never zeroized -- TOTP secret persists in memory

**File:** `twofa/internal/services/twofaService/setup.go:44-48`
**Issue:** `totp.GenerateSecret()` returns `(raw []byte, base32Secret string, error)`. The `raw` bytes are correctly zeroized via `defer crypto.Zeroize(raw)`, but the `base32Secret` string is a Go immutable string that cannot be zeroized. It is passed to `totp.GenerateProvisioningURI()` and embedded in the returned URI. This means the full TOTP secret remains in memory as a string for the lifetime of the GC generation, violating the project constraint "TOTP-secret NEVER persisted; only exists in memory during verify/setup" and SEC-04. Since Go strings are immutable, the only mitigation is to work with `[]byte` throughout and zeroize after URI construction.
**Fix:** Refactor `totp.GenerateSecret()` to return base32 as `[]byte` instead of `string`, and refactor `GenerateProvisioningURI` to accept `[]byte`. Build the URI using byte slices, then zeroize the base32 bytes after the URI is constructed. Example:

```go
// In totp.go:
func GenerateSecret() ([]byte, []byte, error) {
    secret := make([]byte, 20)
    if _, err := io.ReadFull(rand.Reader, secret); err != nil {
        return nil, nil, fmt.Errorf("%w: %v", ErrSecretGeneration, err)
    }
    encoded := []byte(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret))
    return secret, encoded, nil
}

// In setup.go:
raw, base32Secret, err := totp.GenerateSecret()
if err != nil { ... }
defer crypto.Zeroize(raw)
defer crypto.Zeroize(base32Secret)

uri := totp.GenerateProvisioningURI(base32Secret, email)
```

### CR-02: No validation of MPC node count -- panic on empty config

**File:** `twofa/internal/services/twofaService/setup.go:62-64`
**Issue:** `distributeShares` iterates `shares` (always length 3 from `shamir.Split(raw, 3, 2)`) and indexes into `s.mpcClients[i]`. If the config has fewer than 3 MPC nodes, this causes an index-out-of-bounds panic at runtime. There is no validation anywhere in the chain (config.go, bootstrap.go, NewTwoFAService, or Setup) that `len(mpcClients) == 3`. A misconfigured `config.yaml` would crash the service on the first Setup call.
**Fix:** Add validation in `NewTwoFAService`:

```go
func NewTwoFAService(
    storage Storage,
    sessionStorage SessionStorage,
    mpcClients []MPCClient,
    sharedSecret string,
    mpcTimeout time.Duration,
) *TwoFAService {
    if len(mpcClients) != 3 {
        panic(fmt.Sprintf("twofa: expected exactly 3 MPC clients, got %d", len(mpcClients)))
    }
    // ...
}
```

## Warnings

### WR-01: os.Exit(1) inside goroutine skips all deferred cleanup

**File:** `twofa/cmd/app/main.go:63`
**Issue:** When `grpcServer.Serve(lis)` fails inside the goroutine, `os.Exit(1)` is called. This immediately terminates the process without running any deferred functions (pgStorage.Close(), redisStorage.Close(), mpcConns[i].Close()). This leaves database connections, Redis connections, and gRPC connections unclosed.
**Fix:** Send the error to a channel or signal the quit channel instead:

```go
go func() {
    slog.Info("TwoFA service started", "port", cfg.Server.Port)
    if err := grpcServer.Serve(lis); err != nil {
        slog.Error("gRPC server failed", "error", err)
        quit <- syscall.SIGTERM // trigger graceful shutdown path
    }
}()
```

### WR-02: Insecure gRPC connections to MPC nodes (no TLS)

**File:** `twofa/internal/bootstrap/bootstrap.go:48`
**Issue:** MPC node connections use `insecure.NewCredentials()` -- plaintext gRPC. Share data (encrypted Shamir shares) is transmitted without transport-level encryption. While shares are encrypted at-rest on MPC nodes, they travel over the network in plaintext. An attacker with network access between TwoFA and MPC nodes could intercept shares. With 2-of-3 threshold, intercepting 2 shares allows secret reconstruction.
**Fix:** For production, configure TLS credentials. For development, this is acceptable but should be documented with a TODO:

```go
// TODO: Replace with TLS credentials for production deployment
grpc.WithTransportCredentials(insecure.NewCredentials()),
```

### WR-03: Shared secret transmitted as plaintext in gRPC metadata

**File:** `twofa/internal/bootstrap/bootstrap.go:74`
**Issue:** The `authMetadataInterceptor` sends the shared secret as raw plaintext in the `authorization` metadata header. Combined with WR-02 (no TLS), this secret is visible to any network observer. Even with TLS, sending a raw shared secret (rather than a bearer token or HMAC) means the credential is static and has no expiry or rotation mechanism.
**Fix:** At minimum, prefix with "Bearer " for convention. For production, consider HMAC-based request signing or mTLS instead of a static shared secret.

### WR-04: No foreign key or index on backup_codes.user_id

**File:** `twofa/internal/storage/pgstorage/pgstorage.go:52-56`
**Issue:** The `backup_codes` table has `user_id UUID NOT NULL` but no foreign key constraint to `twofa_records.user_id` and no index on `user_id`. Queries that look up backup codes by user_id (which will happen during verification in Phase 8) will require a full table scan. The lack of a foreign key also means backup codes can exist for users without a twofa_record.
**Fix:**

```sql
CREATE TABLE IF NOT EXISTS backup_codes (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES twofa_records(user_id) ON DELETE CASCADE,
    code_hash VARCHAR(255) NOT NULL,
    is_used BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE INDEX IF NOT EXISTS idx_backup_codes_user_id ON backup_codes(user_id);
```

### WR-05: Setup does not delete old backup codes on re-setup

**File:** `twofa/internal/services/twofaService/setup.go:68-81`
**Issue:** When a user has an existing TwoFA record with `is_enabled=false` (i.e., previously set up but not yet verified, or disabled), the Setup flow skips `CreateTwoFARecord` but still generates and stores new backup codes via `StoreBatchBackupCodes`. This appends new codes without deleting the old ones. After multiple re-setups, a user accumulates stale backup code hashes in the database, and old shares remain on MPC nodes (the new shares overwrite by user_id, but this depends on MPC node implementation).
**Fix:** Before storing new backup codes, delete existing ones:

```go
if existing != nil {
    if err := s.storage.DeleteBackupCodes(ctx, userID); err != nil {
        return "", nil, fmt.Errorf("delete old backup codes: %w", err)
    }
}
```

## Info

### IN-01: Zeroize implementation may be optimized away by compiler

**File:** `twofa/internal/crypto/zeroize.go:5-8`
**Issue:** The simple loop `b[i] = 0` could theoretically be optimized away by an aggressive compiler if the slice is not used after zeroization. While Go's current compiler does not perform this optimization, it is a known concern in secure coding. The `crypto/subtle` package or `runtime.KeepAlive` could provide a more robust guarantee.
**Fix:** Consider using `clear(b)` (Go 1.21+) which is less likely to be optimized, or add a compiler hint:

```go
func Zeroize(b []byte) {
    for i := range b {
        b[i] = 0
    }
    runtime.KeepAlive(b)
}
```

### IN-02: Test helper `contains` reimplements `strings.Contains`

**File:** `twofa/internal/services/twofaService/setup_test.go:486-497`
**Issue:** The `contains` and `searchSubstring` helper functions are manual reimplementations of `strings.Contains` from the standard library. Using the standard library function would be clearer and less error-prone.
**Fix:** Replace with `strings.Contains`:

```go
import "strings"
// Remove contains() and searchSubstring() functions
// Replace all calls: contains(uri, "otpauth://totp/") -> strings.Contains(uri, "otpauth://totp/")
```

### IN-03: Config does not validate required fields

**File:** `twofa/config/config.go:62-74`
**Issue:** The `Load` function parses YAML but does not validate that required fields are present (e.g., `Server.Port`, `Database.DSN`, `MPCNodes` having exactly 3 entries, `SharedSecret` being non-empty). A missing or zero-value config field would cause silent misbehavior (e.g., listening on port 0, empty shared secret).
**Fix:** Add a `Validate()` method:

```go
func (c *Config) Validate() error {
    if c.Server.Port == 0 {
        return fmt.Errorf("server.port is required")
    }
    if c.Database.DSN == "" {
        return fmt.Errorf("database.dsn is required")
    }
    if len(c.MPCNodes) != 3 {
        return fmt.Errorf("exactly 3 mpc_nodes required, got %d", len(c.MPCNodes))
    }
    if c.SharedSecret == "" {
        return fmt.Errorf("shared_secret is required")
    }
    return nil
}
```

---

_Reviewed: 2026-04-12T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
