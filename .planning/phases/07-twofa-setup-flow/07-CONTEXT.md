# Phase 7: TwoFA Setup Flow - Context

**Gathered:** 2026-04-12
**Status:** Ready for planning

<domain>
## Phase Boundary

Orchestrate 2FA setup: generate TOTP secret, Shamir split (2-of-3), distribute shares in parallel to 3 MPC nodes via gRPC, generate 10 backup codes (bcrypt-hashed), return provisioning URI. Includes rollback on partial MPC failure, secret zeroization, and duplicate setup prevention. All crypto primitives (Shamir, TOTP) and MPC node service already exist from Phases 4-6.

</domain>

<decisions>
## Implementation Decisions

### MPC Node Communication
- **D-01:** Parallel share distribution — 3 goroutines via `errgroup` with shared context. All 3 StoreShare calls execute concurrently. If any fails, context is cancelled and remaining calls abort.
- **D-02:** Compensating delete on partial failure — if any StoreShare fails, call DeleteShare(user_id) on ALL 3 nodes (idempotent per Phase 6 D-08). Ensures no orphaned shares remain in any node.
- **D-03:** gRPC clients to MPC nodes created at TwoFA service startup, configured via `config.yaml` `mpc_nodes` array (3 entries with `addr` and `shared_secret`). Shared secret sent in gRPC metadata "authorization" header per Phase 6 D-05.
- **D-04:** Timeout per MPC call — use context with timeout from config (default 5s). Setup fails with `codes.Internal` if any node times out.

### Backup Codes
- **D-05:** Format `xxxx-xxxx` — 8 digits split by hyphen. Generated via `crypto/rand`, each half is 4 random digits (0000-9999). 10 codes per setup.
- **D-06:** Codes bcrypt-hashed (cost=12) before storage in `backup_codes` table. Plaintext codes returned to user in Setup2FAResponse only once — never stored or logged.
- **D-07:** On comparison (Phase 8), strip hyphen before bcrypt check. Normalize input: remove hyphens, spaces, leading zeros preserved.

### Secret Lifecycle & Zeroization
- **D-08:** `defer zeroize(secret)` immediately after `totp.GenerateSecret()` call. Zeroize function: loop over `[]byte`, set each to 0. Guarantees cleanup on success, error, and panic paths.
- **D-09:** Shares (`[]Share`) also zeroized after distribution — `defer` for each share's `Data` field. Secret must not survive in any form after setup completes.
- **D-10:** Zeroize utility function in `twofa/internal/crypto/` package — shared between setup (Phase 7) and verify (Phase 8).

### Storage & Interfaces
- **D-11:** Add `email` field to `Setup2FARequest` proto message. Gateway/client passes email alongside user_id. No cross-service gRPC dependency on Auth.
- **D-12:** Duplicate setup prevention — check `twofa_records` for existing record with `is_enabled=true`. If found, return `codes.AlreadyExists`. User must Disable2FA first, then re-setup.
- **D-13:** Storage interface methods for Phase 7:
  - `CreateTwoFARecord(ctx, userID string) error` — insert into twofa_records (is_enabled=false initially)
  - `GetTwoFARecord(ctx, userID string) (*TwoFARecord, error)` — for duplicate check
  - `StoreBatchBackupCodes(ctx, userID string, codeHashes []string) error` — bulk insert 10 hashed codes
- **D-14:** TwoFARecord initially created with `is_enabled=false`. Transitions to `true` only on first successful verification (Phase 8, 2FA-04).

### Test Strategy
- **D-15:** Comprehensive tests (~20+) for  defense:
  - Setup happy path: secret generated, split into 3 shares, all stored, URI + 10 backup codes returned
  - Partial MPC failure: node 2 fails → compensating delete called on all nodes → error returned
  - All MPC nodes fail: error returned, no shares persisted
  - Duplicate setup: is_enabled=true → AlreadyExists error
  - Backup code format: regex match `^\d{4}-\d{4}$`, all 10 unique
  - Zeroization: secret bytes zeroed after setup (check via pointer)
  - Backup code hashing: plaintext != stored hash, bcrypt.CompareHashAndPassword works

### Claude's Discretion
- Exact errgroup pattern and error aggregation
- MPC gRPC client wrapper/pool design (connection management)
- Whether to use a separate `MPCClient` interface or call gRPC stubs directly
- Kafka audit event structure for `2fa.setup_started`, `2fa.setup_completed`, `2fa.setup_failed`
- Prometheus metric labels for setup operations
- Internal helper decomposition within Setup method

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Architecture & Structure
- `CLAUDE.md` — Full project spec, TOTP never persisted, Shamir 2-of-3, zeroization rules, no secret logging
- `workspace/02 - Services/TwoFA Service.md` — TwoFA API, orchestration flow, MPC coordination

### Requirements
- `.planning/REQUIREMENTS.md` — 2FA-01, 2FA-02, 2FA-08, SEC-04

### Crypto Packages (implemented in Phases 4-5)
- `twofa/internal/crypto/shamir/shamir.go` — `Split(secret, n, threshold) ([]Share, error)`, `Combine(shares) ([]byte, error)`
- `twofa/internal/crypto/totp/totp.go` — `GenerateSecret() ([]byte, string, error)`, `GenerateProvisioningURI(secret, email) string`

### MPC Node Service (implemented in Phase 6)
- `mpc/api/mpc_api/mpc.proto` — StoreShare, RetrieveShare, DeleteShare RPC definitions
- `mpc/internal/middleware/interceptors.go` — Shared secret auth interceptor pattern

### Prior Phase Contexts
- `.planning/phases/04-shamir-secret-sharing/04-CONTEXT.md` — Shamir API design, Share struct (Index byte, Data []byte)
- `.planning/phases/05-totp-implementation/05-CONTEXT.md` — TOTP API design, GenerateSecret returns raw+base32
- `.planning/phases/06-mpc-node-service/06-CONTEXT.md` — MPC storage, encryption, idempotent DeleteShare, auth interceptor

### Existing TwoFA Scaffolding (Phase 1)
- `twofa/internal/services/twofaService/twofa_service.go` — Service struct, Storage/SessionStorage interfaces (empty, to be filled)
- `twofa/internal/api/twofa_service_api/twofa_service_api.go` — Service interface (empty), TwoFAServiceAPI struct
- `twofa/internal/api/twofa_service_api/setup.go` — Setup2FA handler stub (returns Unimplemented)
- `twofa/internal/storage/pgstorage/pgstorage.go` — PGStorage with twofa_records + backup_codes tables
- `twofa/internal/bootstrap/bootstrap.go` — DI wiring (PGStorage, RedisStorage, TwoFAService, API, gRPC server)
- `twofa/config/config.go` — Config with MPCNodes array, SharedSecret

### Proto Definitions
- `twofa/api/twofa_api/twofa.proto` — Setup2FA RPC (needs email field added to request)
- `twofa/api/models/models.proto` — TwoFARecord, BackupCode messages

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `twofa/internal/crypto/shamir/` — Split/Combine fully implemented and tested (Phase 4)
- `twofa/internal/crypto/totp/` — GenerateSecret, GenerateOTP, ValidateOTP, GenerateProvisioningURI implemented (Phase 5)
- `twofa/internal/storage/pgstorage/pgstorage.go` — Tables `twofa_records` and `backup_codes` already created via initTables
- `twofa/config/config.go` — MPCNodeConfig with Addr field, SharedSecret for MPC auth
- `twofa/internal/bootstrap/bootstrap.go` — DI wiring pattern established

### Established Patterns
- One file per gRPC method in handler directory (setup.go, verify.go, disable.go, status.go)
- Service struct with constructor accepting storage + session storage
- Interface-based DI with minimock for mock generation
- Domain errors in dedicated package (pattern from auth service)
- gRPC error code mapping in handler layer

### Integration Points
- TwoFAService needs: MPC gRPC clients (3x), crypto packages (shamir, totp)
- Storage interface needs: CreateTwoFARecord, GetTwoFARecord, StoreBatchBackupCodes methods
- Bootstrap needs: MPC gRPC client creation, inject into TwoFAService
- Proto needs: email field in Setup2FARequest, regenerate pb code
- Setup handler: validate request, delegate to service, map errors

</code_context>

<specifics>
## Specific Ideas

- Backup code format `xxxx-xxxx` — 8 digits with hyphen for readability, strip hyphen on validation
- `defer zeroize(secret)` pattern — consistent with security-first approach for 
- errgroup for parallel MPC calls — standard Go concurrency pattern, good for  demonstration
- MPC gRPC clients created once at startup via bootstrap, not per-request
- Share indices map to MPC node indices: share.Index=1 → node[0], share.Index=2 → node[1], share.Index=3 → node[2]

</specifics>

<deferred>
## Deferred Ideas

- **OTP verification flow** — Phase 8 handles Verify2FA (retrieve 2 shares, combine, validate TOTP)
- **Rate limiting** — Phase 8 implements 5-attempt-per-5-min via Redis
- **Disable 2FA** — Phase 8 handles Disable2FA (verify + delete shares + cleanup)
- **2FA status check** — Phase 8 implements Get2FAStatus
- **Kafka audit events** — Phase 9 adds full audit instrumentation
- **Prometheus metrics** — Phase 9 adds metrics

None — discussion stayed within phase scope

</deferred>

---

*Phase: 07-twofa-setup-flow*
*Context gathered: 2026-04-12*
