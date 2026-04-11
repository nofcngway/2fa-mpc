# Codebase Concerns

**Analysis Date:** 2026-04-11

## Critical Implementation Risks

### Shamir Secret Sharing (Custom GF(256) Implementation)
- **Issue**: Custom cryptographic implementation from scratch with no external audit
- **Files**: `twofa/internal/services/twofaService/shamir/` (when implemented)
- **Impact**: Incorrect GF(256) arithmetic (especially multiplication/division via log/exp tables) compromises the entire 2FA system. Even subtle bugs in Lagrange interpolation destroy threshold security
- **Risk Level**: CRITICAL
- **Fix approach**: 
  - Implement extensive unit tests covering all GF(256) operations before any production use
  - Test vectors: verify against reference implementations (e.g., ssss.readthedocs.io)
  - Test recovery with invalid/corrupted shares to ensure they don't accidentally reconstruct
  - Consider getting external cryptographic review before deployment
  - Tests must cover: all GF(256) ops (add/mul/div/inv), split/combine with all 3 combinations of 2 shares, edge cases (empty secret, 1-byte secret, 20-byte TOTP secret)

### TOTP Secret Lifecycle - Memory Safety
- **Issue**: TOTP-secret (20 bytes) must be generated, split, zeroized, but never persisted as plaintext. Memory remains vulnerable to core dumps, debugging tools, and cold-start attacks
- **Files**: `twofa/internal/services/twofaService/` (when implemented)
- **Impact**: Plaintext secret exposure if memory zeroization is skipped or incomplete. Rate limiting failure allows brute-force OTP codes (10^6 combinations in 30 seconds)
- **Risk Level**: HIGH
- **Fix approach**:
  - Implement explicit memory zeroization for secret buffers after Shamir split (use `golang.org/x/crypto/subtle` patterns or manual overwrite)
  - Verify no TOTP-secret strings are logged (implement code review + static analysis)
  - Document exactly which functions hold the secret in memory at each step
  - Rate limiter must be non-bypassable: store per-user counter in Redis with atomic increments, check BEFORE verification attempt

### JWT RS256 Key Management
- **Issue**: Private key must be loaded from configuration without being logged or exposed in error messages. Key rotation requires careful handling
- **Files**: `auth/internal/services/authService/jwt.go`, `auth/config/config.go` (when implemented)
- **Impact**: Private key compromise enables forging arbitrary tokens for any user
- **Risk Level**: HIGH
- **Fix approach**:
  - Load RSA private key from file (protected by filesystem permissions, not in YAML)
  - Never log the key value; only log "private key loaded" message
  - Error messages must not include key material: use `errors.New("key load failed")` not formatted strings with key
  - Public key should be served at a well-known endpoint for other services to validate tokens
  - Document key rotation procedure (old key must still validate in-flight tokens for 15+ minutes after rotation)

### AES-256-GCM Encryption at Rest
- **Issue**: Each MPC node has a unique encryption key, but nonce generation via `crypto/rand` is non-deterministic. Reusing same key+nonce with different plaintext breaks GCM security
- **Files**: `mpc/internal/crypto/aes.go`, `mpc/internal/storage/pgstorage/` (when implemented)
- **Impact**: Complete compromise of encrypted shares in MPC node PostgreSQL if nonce collision occurs or if same share is re-encrypted
- **Risk Level**: HIGH
- **Fix approach**:
  - Strictly enforce: nonce is cryptographically random (12 bytes from `crypto/rand`), unique per encryption operation
  - Store nonce alongside ciphertext (this is standard practice, not a vulnerability)
  - NEVER re-encrypt the same plaintext under same key (each Shamir split call gets new share, not re-encryption of old)
  - Test: verify nonce diversity across 1000+ encrypt operations
  - Do NOT use deterministic nonce derivation (e.g., hash-based) — only `crypto/rand`

### Password Validation - Sequence Detection
- **Issue**: Detecting "4+ characters in sequence" requires checking ASCII ordering, keyboard layouts, and reverse patterns. Incorrect implementation allows weak sequences through
- **Files**: `auth/internal/services/authService/password_validation.go` (when implemented)
- **Impact**: Users can set weak passwords like "Password1!" (0 sequences detected due to bug), defeating password policy
- **Risk Level**: MEDIUM
- **Fix approach**:
  - Implement separate checker for each sequence type: numeric ASCII (0-9), alphabetic ASCII (a-z, case-insensitive), keyboard rows (qwerty, asdf, zxcv)
  - For each sequence, check both forward and reverse
  - Include test cases at boundaries: 3 chars → PASS, 4 chars → FAIL, 5 chars → FAIL
  - Test reverse sequences: "dcba", "4321", "yrewq"
  - Reject both uppercase and lowercase matches: "ABCD" is same sequence as "abcd"

### Rate Limiting on 2FA Verification
- **Issue**: Specification requires "5 attempts per 5 minutes per user_id" but implementation must handle distributed requests across multiple Gateway instances
- **Files**: `twofa/internal/services/twofaService/` (when implemented)
- **Impact**: Brute-force OTP codes (10^6 possibilities) becomes feasible if rate limiter is bypassable
- **Risk Level**: HIGH
- **Fix approach**:
  - Store counter in Redis (shared across all gateways)
  - Key format: `2fa:verify:{user_id}` with value = attempt count, TTL 5 minutes
  - Check counter BEFORE attempting verification (not after)
  - Check and increment atomically: use Redis INCR or Lua script to prevent race conditions
  - Return same error message for "rate limited" and "invalid code" (timing attack mitigation)
  - Reset counter on successful verification

## Design Gaps

### TOTP Provisioning URI Security
- **Issue**: Provisioning URI contains base32-encoded secret and is returned in Setup2FA response. If HTTP/TLS is compromised, secret is exposed before user scans QR code
- **Files**: `twofa/internal/services/twofaService/totp/` (when implemented)
- **Impact**: Attacker intercepts provisioning URI → extracts secret → can forge valid OTP codes, completely bypassing 2FA setup
- **Risk Level**: MEDIUM
- **Fix approach**:
  - This is a protocol-level issue, not implementation: provisioning URI MUST be transmitted only over HTTPS with strong TLS
  - Consider returning only QR code URI (encrypted), not plaintext secret
  - Document to frontend: if Setup2FA response is logged/cached, 2FA is compromised
  - Add warning to API response: "Keep provisioning URI secure — anyone with this URI can forge OTP codes"

### TOTP Backup Codes Implementation Missing
- **Issue**: Specification mentions backup codes (10 hashed codes) but no implementation approach is documented
- **Files**: `twofa/internal/storage/pgstorage/` (when implemented — backup_codes table)
- **Impact**: If TOTP device is lost and no backup codes exist, user loses all 2FA access; if backup codes are weak/predictable, user can be locked out by attacker
- **Risk Level**: MEDIUM
- **Fix approach**:
  - Generate 10 random backup codes (12 alphanumeric each) during Setup2FA
  - Store only bcrypt hashes in database (same as password hashing)
  - Return codes to user ONLY once, at setup time — user must save them
  - Implement DisableBackupCode endpoint to prevent reuse (mark used codes in DB)
  - Require user to confirm they've saved backup codes before Setup2FA completes
  - Document: backup codes are one-time use, not 10-use pool

### MPC Node Authorization
- **Issue**: Specification requires "shared secret via gRPC metadata", but implementation approach is not detailed. Shared secret must be managed securely
- **Files**: `mpc/internal/middleware/interceptors.go` (when implemented)
- **Impact**: If shared secret is hardcoded, logged, or exposed in error messages, any compromised service gains access to all shares
- **Risk Level**: HIGH
- **Fix approach**:
  - Load shared secret from environment variables (separate for each node)
  - Each MPC node has different shared secret (NODE_ID=1 → SECRET_1, NODE_ID=2 → SECRET_2, NODE_ID=3 → SECRET_3)
  - gRPC interceptor checks "authorization" metadata header: expect exactly `Bearer {secret}` format
  - Never log the shared secret, only "authorization check passed/failed"
  - Error messages must not leak secret value: return `Unauthenticated` status without details
  - Document: shared secrets are per-node and must be rotated without downtime (dual-validation period)

### Cross-Service Communication Security
- **Issue**: TwoFA → MPC nodes communication is gRPC-only, but if network is compromised or service runs outside Kubernetes, plaintext gRPC is vulnerable
- **Files**: `twofa/internal/api/twofa_service_api/` (when implemented)
- **Impact**: Attacker on network can intercept shares flowing from TwoFA to MPC nodes
- **Risk Level**: MEDIUM
- **Fix approach**:
  - Require mTLS between TwoFA and all MPC nodes (gRPC with client certificates)
  - Each service has certificate, MPC nodes validate TwoFA certificate
  - Document mTLS setup in `monitoring/` (Kubernetes-style certificate rotation)
  - Alternative if mTLS is not available: encrypt shares before transmission (double encryption: Shamir result → AES-256-GCM)

## Testing Coverage Gaps

### Cryptography Testing
- **What's not tested**: Shamir combine with wrong share indices, corrupted share data recovery, zero-valued shares, non-random coefficients
- **Files**: `twofa/internal/services/twofaService/shamir/shamir_test.go` (when implemented)
- **Risk**: Silent failures or incorrect reconstructions go undetected until production
- **Priority**: CRITICAL
- **Action items**:
  - Test combine with (share[0], share[1]), (share[0], share[2]), (share[1], share[2]) separately
  - Test combine with 1 share only → must fail, not accidentally reconstruct
  - Test with all-zero secret (edge case)
  - Test GF(256) operations with identity elements (0, 1) and edge values (254, 255)

### Error Path Testing
- **What's not tested**: Kafka producer failure handling, PostgreSQL connection loss during share storage, gRPC deadline exceeded, partial share failure (1 of 3 MPC nodes down)
- **Files**: All service repositories and Kafka integration (when implemented)
- **Risk**: System becomes inconsistent (1 share stored, 2 failed) or hangs indefinitely
- **Priority**: HIGH
- **Action items**:
  - Test StoreShare when MPC node 2 is unreachable (circuit breaker logic needed)
  - Test Verify2FA when TwoFA → MPC gRPC call times out (5 second deadline)
  - Test Kafka producer failure doesn't block authentication flow (fire-and-forget semantics)

### Integration Testing
- **What's not tested**: Full flow: Setup2FA → retrieve from all 3 MPC nodes → combine → verify OTP, under various failure modes
- **Files**: Integration test suite (missing)
- **Risk**: System appears to work in unit tests but fails in production under realistic conditions
- **Priority**: HIGH
- **Action items**:
  - Create integration test: Register → Login → Setup2FA → Verify2FA with valid OTP
  - Create failure test: Setup2FA with 1 MPC node down → Verify2FA with different node down → must still succeed
  - Test Redis failure during token refresh
  - Test Kafka downtime doesn't block auth/2fa operations

## Known Limitations (By Design)

### TOTP Synchronization Window
- **Issue**: Specification allows ±1 time window (±30 seconds) for clock skew, but no explicit handling of clock drift scenarios
- **Files**: `twofa/internal/services/twofaService/totp/` (when implemented)
- **Impact**: User with clock 60+ seconds ahead/behind cannot login, but 30-second drift is accepted
- **Severity**: LOW (acceptable per RFC 6238)
- **Mitigation**: Document to users: keep device time synchronized. Consider server-side time sync audit.

### Shamir Share Recovery Failure
- **Issue**: If 2+ MPC nodes are down during Verify2FA, recovery is impossible (requires 2-of-3 shares)
- **Files**: `twofa/internal/services/twofaService/` (when implemented)
- **Impact**: User cannot authenticate with 2FA if 2+ MPC nodes are down
- **Severity**: MEDIUM (operational requirement, not security)
- **Mitigation**: Implement MPC node health checks and failover. Document operational SLA: "All 3 MPC nodes must be available 99.9% of the time".

## Security-Specific Concerns

### Logging of Sensitive Data
- **Issue**: Specification prohibits logging secrets, passwords, shares, encryption keys. Implementation must be verified to enforce this
- **Files**: All service files implementing logging (when implemented)
- **Risk**: Accidental logging of secrets in debug logs, error messages, Kafka events
- **Priority**: CRITICAL
- **Verification approach**:
  - Use static analysis: grep for logger calls that include password, secret, key, share variables
  - Log only non-sensitive metadata: user_id (UUID ok), operation name, status, latency
  - Test: set breakpoint in logger, ensure no sensitive data is passed
  - Kafka events must log: user_id, operation, timestamp; NOT share_data or secret material

### Password Storage & Hashing
- **Issue**: Specification requires bcrypt cost=12, but no guidance on pepper/salt configuration
- **Files**: `auth/internal/services/authService/` (when implemented)
- **Risk**: Bcrypt with cost=12 takes ~100ms per hash. If user creates 1000 accounts rapidly, hash generation becomes bottleneck or attacker might use timing to detect hash cost mismatches
- **Priority**: LOW
- **Approach**:
  - Bcrypt automatically uses salt (16 bytes) per call — no additional pepper needed
  - Cost=12 is industry standard (~100ms) — acceptable for registration/password change, not for every login (JWT used instead)
  - Monitor hashing latency: if > 500ms, reduce concurrent registration attempts

### Token Revocation Gaps
- **Issue**: Access tokens (15-minute TTL) cannot be revoked before expiry if user account is compromised. Refresh tokens can be revoked (deleted from Redis) but access tokens in-flight are still valid
- **Files**: `auth/internal/services/authService/` (when implemented)
- **Risk**: If user account is compromised, attacker with access token can perform actions for up to 15 minutes
- **Severity**: LOW (acceptable trade-off for performance; 15 minutes is short window)
- **Mitigation**: 
  - Implement token blacklist (Redis set of revoked access tokens with TTL 15 minutes) for emergency revocation
  - On Logout, delete refresh token AND invalidate all active access tokens (optional, expensive)
  - For critical operations (password change), require fresh re-authentication (new login)

## Scaling & Performance Concerns

### Redis Dependency for Rate Limiting & Sessions
- **Issue**: Rate limiting and refresh tokens are stored in Redis. Single Redis instance is a bottleneck; Redis failure causes complete auth/2FA lockout
- **Files**: `gateway/internal/` and `auth/internal/storage/redisstorage/` (when implemented)
- **Impact**: Redis cluster failover or restart → all users unable to login/verify 2FA (total outage)
- **Severity**: MEDIUM
- **Recommendation**:
  - Document minimum availability: Redis must have replication + sentinel for HA
  - Implement circuit breaker in gRPC calls: if Redis unreachable, fall back to in-memory cache for rate limiting (less accurate but service continues)
  - Monitor Redis latency; if > 100ms, increase concurrent connections to Redis pool

### MPC Node Latency Impact on User Experience
- **Issue**: Verify2FA requires sequential gRPC calls to 2 MPC nodes (5-second timeout each). Total latency: ~500-1000ms per verification attempt
- **Files**: `twofa/internal/services/twofaService/` (when implemented)
- **Impact**: User experience degradation; on high-latency networks, timeout is reached
- **Severity**: LOW-MEDIUM
- **Recommendation**:
  - Parallelize MPC node calls: issue both calls concurrently, wait for first 2 responses (no waiting for slow node)
  - Implement request pipelining: combine share retrieval with combine operation on same gRPC stream
  - Cache MPC node health: avoid timeouts by pre-checking which nodes are alive

### PostgreSQL Concurrent Write Bottleneck
- **Issue**: All services write audit logs to PostgreSQL. High-throughput scenarios (1000+ requests/second) cause write saturation
- **Files**: All services' storage implementations (when implemented)
- **Impact**: Audit log writes fail or are delayed; database connection pool exhaustion
- **Severity**: LOW (audit is non-critical path)
- **Recommendation**:
  - Async audit logging: write to Kafka first, let separate consumer batch insert to PostgreSQL
  - Use pgx batch API for bulk inserts (not single-row inserts)
  - Monitor query latency; if > 100ms, implement read replicas for queries

## Fragile Areas

### Shamir Split/Combine Correctness
- **Why fragile**: Custom cryptography implementation with no external audit. Single arithmetic bug invalidates entire protocol
- **Files**: `twofa/internal/services/twofaService/shamir/gf256.go`, `shamir.go` (when implemented)
- **Safe modification**: 
  - Never modify GF(256) operations (add/mul/div) without understanding finite field theory
  - Add test cases BEFORE modifying, AFTER modification verify all tests still pass
  - Use reference implementation (e.g., Python shamir package) to generate test vectors
  - Consider external code review for any changes

### API Gateway REST-to-gRPC Translation
- **Why fragile**: Protocol translation layer is custom code. Bugs here cause field mismatches, type coercion errors, or lost data
- **Files**: `gateway/internal/api/` (when implemented)
- **Safe modification**:
  - Ensure request field names match proto message field names exactly (camelCase ↔ snake_case mapping)
  - Test each endpoint independently: call REST endpoint, verify gRPC call received correct data
  - Add integration tests for each REST endpoint
  - Use code generation if possible (e.g., grpc-gateway) instead of manual translation

### JWT Token Validation
- **Why fragile**: Incorrect validation logic allows forged tokens. Common mistakes: not checking signature, not checking expiry, accepting expired tokens
- **Files**: `auth/internal/services/authService/validate.go` and all consumers (when implemented)
- **Safe modification**:
  - Use standard library (`github.com/golang-jwt/jwt/v5`) for token parsing and validation
  - Always verify: signature (using public key), expiry (iat + TTL > now), subject (user_id format)
  - Never accept token with invalid signature or expired claim
  - Test: pass malformed tokens, expired tokens, tokens signed with wrong key — all must be rejected

## Missing Critical Features

### MPC Node Backup & Recovery
- **Issue**: No backup strategy documented for MPC node shares. If PostgreSQL on MPC node 1 is corrupted, that share is permanently lost
- **Impact**: User loses 2FA access if 2+ nodes are non-recoverable
- **Blocks**: Operational deployment
- **Approach**:
  - Implement automated PostgreSQL backups (daily snapshots to off-site storage)
  - Document recovery procedure: restore PostgreSQL backup, restart node
  - Monitor: ensure all 3 nodes have recent backups

### Graceful Shutdown Implementation
- **Issue**: Specification requires graceful shutdown for all services, but no implementation is documented
- **Files**: `*/cmd/app/main.go` (when implemented)
- **Impact**: Abrupt shutdown can leave connections open, uncommitted transactions, or orphaned Kafka messages
- **Approach**:
  - Implement signal handling (SIGTERM): 30-second drain period before shutdown
  - During drain: stop accepting new requests, finish in-flight requests
  - Close connections: gRPC listener, PostgreSQL pool, Redis, Kafka producer
  - Implement HTTP health check endpoint that returns 503 during drain
  - Document: Docker/Kubernetes should wait for graceful shutdown before force-killing

### gRPC Health Check Protocol
- **Issue**: Specification requires Health Check Protocol but no implementation is documented
- **Files**: Each service's `cmd/app/main.go` (when implemented)
- **Impact**: Kubernetes/Docker can detect dead services and restart them
- **Approach**:
  - Import `google.golang.org/grpc/health/grpc_health_v1`
  - Register health service in gRPC server
  - Implement check for each service dependency: PostgreSQL, Redis, MPC nodes (if applicable)
  - Return SERVING only if all dependencies are healthy

## Dependency at Risk

### `golang.org/x/crypto` Version Pinning
- **Issue**: Cryptographic library must be pinned to known-good version. No version constraints documented
- **Impact**: Automatic updates via `go get -u` could introduce breaking changes or bugs in crypto code
- **Recommendation**: 
  - Pin exact version in `go.mod`: `golang.org/x/crypto v0.X.Y`
  - Test all crypto operations on upgrade (Shamir, AES, bcrypt, TOTP)
  - Subscribe to golang security mailing list for crypto package updates

### Custom Shamir Implementation Instead of Established Library
- **Issue**: Decision (ADR-002) to implement Shamir from scratch, not using battle-tested libraries
- **Impact**: Maintainability, security audit difficulty, risk of cryptographic bugs
- **Migration plan** (if risk becomes unacceptable):
  - Consider migration to `github.com/shamirsecretsharing/go-shamir` if available and trustworthy
  - Would require extensive testing to ensure binary-compatible share format
  - Document in ADR why migration was done (if ever attempted)

## Summary of Priorities

| Category | Count | Status |
|----------|-------|--------|
| Critical | 5 | Shamir GF(256), TOTP secrets, JWT keys, AES nonces, Rate limiting |
| High | 6 | Password validation, MPC auth, TLS/mTLS, Memory zeroization, Backup codes, Graceful shutdown |
| Medium | 7 | TOTP URI security, MPC node recovery, Scaling, Logging verification, Token revocation |
| Low | 4 | Clock sync, Share recovery limitations, Hash cost, Code fragility |

**Recommended next steps before any production deployment:**
1. Implement and test all cryptographic modules with external review
2. Implement comprehensive test suite (unit + integration) for all critical paths
3. Implement rate limiting with Redis atomicity guarantees
4. Implement graceful shutdown and health checks
5. Conduct security audit by external cryptography expert
6. Set up monitoring and alerting for all MPC nodes
7. Document operational procedures (backups, key rotation, node recovery)

---

*Concerns audit: 2026-04-11*
