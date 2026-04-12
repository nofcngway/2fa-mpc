---
phase: 06-mpc-node-service
reviewed: 2026-04-12T12:00:00Z
depth: standard
files_reviewed: 17
files_reviewed_list:
  - mpc/cmd/app/main.go
  - mpc/internal/api/mpc_service_api/delete_share.go
  - mpc/internal/api/mpc_service_api/retrieve_share.go
  - mpc/internal/api/mpc_service_api/store_share.go
  - mpc/internal/bootstrap/bootstrap.go
  - mpc/internal/middleware/interceptors.go
  - mpc/internal/middleware/interceptors_test.go
  - mpc/internal/services/mpcService/delete_share.go
  - mpc/internal/services/mpcService/delete_share_test.go
  - mpc/internal/services/mpcService/encrypt.go
  - mpc/internal/services/mpcService/encrypt_test.go
  - mpc/internal/services/mpcService/mpc_service.go
  - mpc/internal/services/mpcService/retrieve_share.go
  - mpc/internal/services/mpcService/retrieve_share_test.go
  - mpc/internal/services/mpcService/store_share.go
  - mpc/internal/services/mpcService/store_share_test.go
  - mpc/internal/storage/pgstorage/share.go
findings:
  critical: 1
  warning: 4
  info: 2
  total: 7
status: issues_found
---

# Phase 6: Code Review Report

**Reviewed:** 2026-04-12T12:00:00Z
**Depth:** standard
**Files Reviewed:** 17
**Status:** issues_found

## Summary

The MPC Node service implements share storage, retrieval, and deletion with AES-256-GCM encryption at rest, gRPC API handlers, shared-secret authentication, and PostgreSQL persistence. The code follows Clean Architecture conventions, uses parameterized SQL queries, constant-time secret comparison, and crypto/rand for nonce generation. Overall quality is solid. One critical security issue exists around encryption key lifecycle. Several warnings address missing validation, plaintext data not being zeroed after encryption, and a potential integer overflow.

## Critical Issues

### CR-01: Encryption key stored in config.yaml as plaintext string -- no zeroization

**File:** `mpc/internal/bootstrap/bootstrap.go:33`
**Issue:** The encryption key is read from `cfg.Node.EncryptionKey` (a `string`) and converted to `[]byte` via `[]byte(cfg.Node.EncryptionKey)`. Go strings are immutable and cannot be zeroed from memory. The original `Config.Node.EncryptionKey` string persists in memory for the lifetime of the process. Per CLAUDE.md: "after split/combine in memory -- immediately zeroize." While the `MPCService` struct holds `encryptionKey []byte` (which could theoretically be zeroed on shutdown), the source `string` in the `Config` struct can never be zeroed.
**Fix:** Store the encryption key in config as a hex or base64-encoded value, decode it directly into a `[]byte` in the config loader, and clear the raw config field immediately after. Alternatively, load the key from an environment variable into `[]byte` directly, avoiding the string intermediary:
```go
// In config.go -- store as []byte, decode from hex
type NodeConfig struct {
    ID               int    `yaml:"id"`
    EncryptionKeyHex string `yaml:"encryption_key"`
    encryptionKey    []byte // unexported, populated after load
}

// After loading config:
key, err := hex.DecodeString(cfg.Node.EncryptionKeyHex)
if err != nil {
    return nil, fmt.Errorf("invalid encryption_key hex: %w", err)
}
cfg.Node.encryptionKey = key
// Zero the hex string source (best-effort for strings)
```

## Warnings

### WR-01: Decrypted plaintext not zeroed after use in RetrieveShare

**File:** `mpc/internal/services/mpcService/retrieve_share.go:22-31`
**Issue:** The `RetrieveShare` method decrypts the share into a `plaintext` byte slice and returns it to the caller. Per project security rules, sensitive data (shares, secrets) should be zeroed from memory after use. The decrypted share data is returned directly with no mechanism for the caller or service to zeroize it. Similarly in `StoreShare` (store_share.go:18), the incoming `shareData` parameter is never zeroed after encryption.
**Fix:** Add a `zeroize` helper and document that callers must zero returned share data, or zero the input data after encryption in `StoreShare`:
```go
func zeroize(b []byte) {
    for i := range b {
        b[i] = 0
    }
}

// In StoreShare, after encrypt succeeds:
defer zeroize(shareData)
```

### WR-02: No validation of shared_secret configuration value

**File:** `mpc/internal/bootstrap/bootstrap.go:53`
**Issue:** `cfg.SharedSecret` is passed directly to `middleware.AuthInterceptor` without validating that it is non-empty or meets a minimum length. If `shared_secret` is omitted from config.yaml, the empty string is used as the authentication secret. The `AuthInterceptor` does check for empty authorization header values (interceptors.go:32), but if both server and client have empty secrets, authentication is effectively bypassed because `subtle.ConstantTimeCompare([]byte(""), []byte(""))` returns 1.
**Fix:** Validate `SharedSecret` during config loading or in bootstrap:
```go
if cfg.SharedSecret == "" {
    return nil, fmt.Errorf("shared_secret must be configured")
}
if len(cfg.SharedSecret) < 32 {
    return nil, fmt.Errorf("shared_secret too short (min 32 chars)")
}
```

### WR-03: Missing Kafka audit event publishing

**File:** `mpc/internal/services/mpcService/store_share.go`, `retrieve_share.go`, `delete_share.go`
**Issue:** Per CLAUDE.md, all services must publish audit events to Kafka containing user_id, operation, and timestamp. The `MPCService` struct has no Kafka producer field, and none of the three operations (StoreShare, RetrieveShare, DeleteShare) publish audit events. The `KafkaConfig` exists in config.go (line 29-32) but is never used.
**Fix:** Add a Kafka producer to the `MPCService` struct and publish events asynchronously in each operation (as done in the auth service pattern). At minimum, add a TODO tracking this as a known gap if it is intentionally deferred to a later phase.

### WR-04: Integer truncation in DeleteShare API handler

**File:** `mpc/internal/api/mpc_service_api/delete_share.go:22`
**Issue:** The service returns `int64` from `DeleteShare`, but the handler casts it to `int32` via `int32(count)`. While practically this will never overflow (a user has at most 3 shares), the silent truncation from int64 to int32 is a code smell and could silently lose data if the underlying query or schema changes.
**Fix:** Either change the protobuf field to `int64`, or add a bounds check:
```go
if count > math.MaxInt32 {
    return nil, status.Error(codes.Internal, "deleted count overflow")
}
return &pb.DeleteShareResponse{DeletedCount: int32(count)}, nil
```

## Info

### IN-01: AES cipher and GCM created on every encrypt/decrypt call

**File:** `mpc/internal/services/mpcService/encrypt.go:11-26`, `encrypt.go:29-46`
**Issue:** Both `encrypt` and `decrypt` create a new `aes.NewCipher` and `cipher.NewGCM` on every invocation. Since the encryption key is constant for the lifetime of the service, the `cipher.AEAD` instance could be created once in `NewMPCService` and reused. This is a minor inefficiency, not a bug -- `cipher.AEAD` from the standard library is safe for concurrent use.
**Fix:** Create the GCM AEAD once during service initialization and store it in the struct.

### IN-02: Config loads from hardcoded relative path "config.yaml"

**File:** `mpc/cmd/app/main.go:21`
**Issue:** The config path `"config.yaml"` is hardcoded. This means the service must always be started from the directory containing config.yaml. This is standard for this project (matches auth service pattern), but worth noting for deployment flexibility.
**Fix:** Accept config path from an environment variable or command-line flag as an optional enhancement.

---

_Reviewed: 2026-04-12T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
