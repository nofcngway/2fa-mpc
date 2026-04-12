# Phase 6: MPC Node Service - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-12
**Phase:** 06-mpc-node-service
**Areas discussed:** Encryption approach, Auth interceptor design, Storage operations, Test strategy

---

## Encryption Approach

### Where should AES-256-GCM encrypt/decrypt live?

| Option | Description | Selected |
|--------|-------------|----------|
| Service layer | MPCService does encrypt before calling storage.CreateShare, decrypt after storage.GetShare. Storage only deals with already-encrypted bytes. Clean separation — storage is pure CRUD. | ✓ |
| Dedicated crypto helper | Separate internal/crypto/aes/ package with Encrypt/Decrypt functions. Reusable pattern consistent with shamir/ and totp/ packages. | |
| You decide | Claude chooses the best approach | |

**User's choice:** Service layer
**Notes:** Clean separation — storage is pure CRUD.

### Should the encryption key be validated on startup?

| Option | Description | Selected |
|--------|-------------|----------|
| Validate and fail fast | Bootstrap checks key length when creating MPCService. Service refuses to start with wrong key size. | ✓ |
| No validation | Trust the config. If key is wrong, encrypt/decrypt will fail at call time. | |
| You decide | Claude chooses | |

**User's choice:** Validate and fail fast
**Notes:** None

### How should corrupted ciphertext be handled during RetrieveShare?

| Option | Description | Selected |
|--------|-------------|----------|
| Return gRPC Internal error | Log the decryption failure (without data), return codes.Internal to caller. TwoFA service sees it as node failure. | ✓ |
| Return specific error | Return a distinct gRPC error code (e.g., DataLoss) so the caller knows the share is corrupted. | |
| You decide | Claude chooses | |

**User's choice:** Return gRPC Internal error
**Notes:** None

---

## Auth Interceptor Design

### Should the interceptor protect all gRPC methods or allow exceptions?

| Option | Description | Selected |
|--------|-------------|----------|
| Protect all methods | Every RPC requires shared secret. Health check excluded automatically. Simplest, most secure. | ✓ |
| Selective protection | Allow some methods to bypass. Requires a whitelist/skip-list. | |

**User's choice:** Protect all methods
**Notes:** None

### How should the interceptor read the shared secret?

| Option | Description | Selected |
|--------|-------------|----------|
| From config (injected) | Interceptor receives the expected shared secret from config at creation time. Matches existing config pattern. | ✓ |
| From closure in bootstrap | Bootstrap creates interceptor as a closure capturing the config value. Functionally the same. | |
| You decide | Claude chooses | |

**User's choice:** From config (injected)
**Notes:** None

---

## Storage Operations

### How should DeleteShare work?

| Option | Description | Selected |
|--------|-------------|----------|
| By user_id only | DELETE FROM shares WHERE user_id = $1. Matches MPC-03 requirement. | ✓ |
| Support both | Two methods: DeleteShareByUser and DeleteShare. More flexible. | |
| You decide | Claude chooses | |

**User's choice:** By user_id only
**Notes:** Matches MPC-03 requirement directly.

### What should RetrieveShare return when no share exists?

| Option | Description | Selected |
|--------|-------------|----------|
| gRPC NotFound error | Standard gRPC convention. Caller knows explicitly. | ✓ |
| Empty response | Return empty/nil share data without error. Less idiomatic. | |
| You decide | Claude chooses | |

**User's choice:** gRPC NotFound error
**Notes:** None

### Should DeleteShare return an error when user has no shares?

| Option | Description | Selected |
|--------|-------------|----------|
| Silent success | Return success even if no rows deleted. Idempotent. Simpler for TwoFA orchestration. | ✓ |
| NotFound error | Return codes.NotFound if no shares existed. Stricter. | |
| You decide | Claude chooses | |

**User's choice:** Silent success
**Notes:** Idempotent design simplifies TwoFA orchestration.

---

## Test Strategy

### What test approach for storage CRUD operations?

| Option | Description | Selected |
|--------|-------------|----------|
| Interface mock tests | Define Storage interface, mock it in service tests. Fast, no DB needed. | ✓ |
| Integration tests with real DB | Tests use real PostgreSQL. Slower but catches SQL bugs. | |
| Both layers | Mock for service + integration for storage. Most thorough. | |
| You decide | Claude chooses | |

**User's choice:** Interface mock tests
**Notes:** None

### Test coverage scope?

| Option | Description | Selected |
|--------|-------------|----------|
| Focused (~15 tests) | Cover each requirement with 2-3 tests each. | |
| Comprehensive (25+ tests) | Full coverage with all edge cases. Maximum for  defense. | ✓ |
| You decide | Claude chooses | |

**User's choice:** Comprehensive (25+ tests)
**Notes:** Maximum coverage for  defense.

---

## Claude's Discretion

- Exact encrypt/decrypt helper function signatures within MPCService
- Error message wording (must not leak internal state per SEC-02)
- Whether to use google.uuid for share ID generation or let PostgreSQL generate
- Kafka audit event publishing structure
- Prometheus metric label design

## Deferred Ideas

- TwoFA → MPC integration — Phase 7
- mTLS between services — v2 requirement (ASEC-01)
- Full Prometheus metrics — Phase 9
- Full Kafka audit events — Phase 9
