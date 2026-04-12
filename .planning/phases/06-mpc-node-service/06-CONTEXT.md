# Phase 6: MPC Node Service - Context

**Gathered:** 2026-04-12
**Status:** Ready for planning

<domain>
## Phase Boundary

Implement the MPC Node service business logic: AES-256-GCM encryption/decryption of share data, PostgreSQL CRUD operations for encrypted shares, gRPC handler implementations (StoreShare, RetrieveShare, DeleteShare), and shared-secret authentication interceptor. All scaffolding (proto, config, models, bootstrap, storage tables) already exists from Phase 1.

</domain>

<decisions>
## Implementation Decisions

### Encryption Approach
- **D-01:** Encryption/decryption lives in the service layer (MPCService). Storage layer is pure CRUD — receives and returns already-encrypted bytes + nonce. Clean separation of concerns.
- **D-02:** Encryption key validated at startup — bootstrap checks key length = 32 bytes when creating MPCService. Service refuses to start with invalid key size. Prevents runtime panics.
- **D-03:** Corrupted ciphertext during RetrieveShare returns gRPC `codes.Internal`. Log the decryption failure context (without share data), caller (TwoFA service) treats it as node failure.

### Auth Interceptor Design
- **D-04:** Shared secret interceptor protects ALL gRPC methods (StoreShare, RetrieveShare, DeleteShare). Health check excluded automatically (registered as separate gRPC service).
- **D-05:** Interceptor receives expected shared secret from config at creation time (injected via constructor). Uses `subtle.ConstantTimeCompare` for timing-safe comparison. Reads client secret from gRPC metadata "authorization" header.

### Storage Operations
- **D-06:** DeleteShare operates by user_id only — `DELETE FROM shares WHERE user_id = $1`. Removes all shares for a user from this node. Matches MPC-03 requirement.
- **D-07:** RetrieveShare returns gRPC `codes.NotFound` when no share exists for given user_id + share_index. Standard gRPC convention.
- **D-08:** DeleteShare returns silent success (no error) when user has no shares to delete. Idempotent — calling delete twice is safe. Simplifies TwoFA orchestration.

### Test Strategy
- **D-09:** Storage tested via interface mocks in service tests. Define Storage interface with CreateShare, GetShare, DeleteSharesByUserID methods. Mock in service unit tests.
- **D-10:** Comprehensive test coverage (25+ tests) for  defense:
  - AES-256-GCM: encrypt/decrypt roundtrip, different nonces per call, wrong key fails, corrupted ciphertext fails, empty plaintext
  - Service: StoreShare happy path, RetrieveShare happy path, DeleteShare happy path, duplicate store rejected, share not found, decrypt failure handling
  - Interceptor: valid secret passes, wrong secret rejected (Unauthenticated), missing metadata rejected, empty secret rejected, constant-time comparison
  - Edge cases: max share data size, zero-length share data, invalid UUID format in request

### Claude's Discretion
- Exact encrypt/decrypt helper function signatures within MPCService
- Error message wording (must not leak internal state per SEC-02)
- Whether to use `google.uuid` for share ID generation or let PostgreSQL generate
- Kafka audit event publishing structure (fire-and-forget pattern)
- Prometheus metric label design

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Architecture & Structure
- `CLAUDE.md` — AES-256-GCM requirement, nonce via crypto/rand, shared secret auth, no secret logging
- `workspace/02 - Services/MPC Node.md` — MPC Node API, storage schema, encryption spec, authorization, Kafka events, Prometheus metrics

### Requirements
- `.planning/REQUIREMENTS.md` — MPC-01, MPC-02, MPC-03, MPC-04, MPC-05, MPC-06

### Existing Code (Phase 1 scaffolding)
- `mpc/internal/services/mpcService/mpc_service.go` — Service struct with encryptionKey and nodeID fields
- `mpc/internal/storage/pgstorage/pgstorage.go` — PGStorage with initTables (shares table schema, UNIQUE constraint)
- `mpc/internal/api/mpc_service_api/` — gRPC handler stubs (StoreShare, RetrieveShare, DeleteShare returning Unimplemented)
- `mpc/internal/middleware/interceptors.go` — LoggingInterceptor (needs auth interceptor added)
- `mpc/internal/models/models.go` — Share domain model (ID, UserID, ShareIndex, EncryptedData, Nonce, CreatedAt)
- `mpc/internal/bootstrap/bootstrap.go` — DI wiring (NewPGStorage, NewMPCService, NewGRPCServer)
- `mpc/config/config.go` — Config with Node.EncryptionKey, Node.ID, SharedSecret fields

### Prior Phase Patterns
- `.planning/phases/04-shamir-secret-sharing/04-CONTEXT.md` — Crypto package pattern, test approach
- `.planning/phases/05-totp-implementation/05-CONTEXT.md` — Consistent crypto patterns

### Security Decisions
- `workspace/04 - Decisions/ADR Log.md` — Architecture decisions

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `mpc/internal/models/models.go` — Share model already defined with correct fields
- `mpc/internal/storage/pgstorage/pgstorage.go` — PGStorage with connection pool and initTables (shares table with UNIQUE constraint on user_id, share_index)
- `mpc/config/config.go` — Config already has Node.EncryptionKey, Node.ID, SharedSecret
- `mpc/internal/bootstrap/bootstrap.go` — Already wires storage → service → API → gRPC server
- `mpc/internal/middleware/interceptors.go` — LoggingInterceptor exists, auth interceptor needs to be added alongside it

### Established Patterns
- One file per gRPC method in `internal/api/mpc_service_api/` (store_share.go, retrieve_share.go, delete_share.go)
- Service struct with constructor accepting storage + config values (NewMPCService)
- pgx connection pool via pgxpool.New
- gRPC interceptor as standalone function in middleware package

### Integration Points
- Bootstrap needs update: chain auth interceptor with logging interceptor in NewGRPCServer
- Service needs: encrypt/decrypt methods + storage CRUD calls
- Storage needs: CreateShare, GetShare, DeleteSharesByUserID methods
- Handlers need: request validation + service delegation + gRPC status code mapping

</code_context>

<specifics>
## Specific Ideas

- AES-256-GCM nonce is 12 bytes, generated fresh via `crypto/rand` for every StoreShare call — never reused
- `crypto/aes` + `cipher.NewGCM` from Go standard library — no external crypto dependencies
- Share ID generated as UUID before storage insertion
- Kafka audit events: `share.stored`, `share.retrieved`, `share.deleted` with user_id, share_index, node_id, timestamp — never share_data or encryption keys

</specifics>

<deferred>
## Deferred Ideas

- **TwoFA → MPC integration** — Phase 7 wires TwoFA service to call MPC nodes for share distribution
- **mTLS between services** — v2 requirement (ASEC-01), currently using shared secret
- **Prometheus metrics** — Phase 9 adds full metrics instrumentation
- **Kafka audit events** — Phase 9 adds full audit event publishing

</deferred>

---

*Phase: 06-mpc-node-service*
*Context gathered: 2026-04-12*
