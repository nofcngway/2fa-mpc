# Codebase Concerns

**Analysis Date:** 2026-04-11

## Project Status

This project is in early planning phase with comprehensive specifications but **no implementation code yet**. Service directories (`auth/`, `gateway/`, `twofa/`, `mpc/`) are empty. Concerns listed below are prospective risks identified from architectural design, specification requirements, and project structure.

---

## Security Concerns

### Critical: Shamir Secret Sharing Implementation Risk

**Risk:** Custom cryptographic implementation of Shamir Secret Sharing in GF(256)

**Impact:** Incorrect arithmetic in GF(256) or flawed polynomial interpolation could allow:
- Secrets recoverable from insufficient shares (threshold 2-of-3 broken)
- Invalid shares accepted as valid
- Data corruption during split/combine cycles

**Files:** `twofa/internal/services/twofaService/shamir/gf256.go`, `twofa/internal/services/twofaService/shamir/shamir.go`

**Current Mitigation:**
- Design specifies unit tests (`shamir_test.go`) for:
  - Split→Combine roundtrip
  - Any 2-of-3 shares recover original
  - 1-of-3 fails to recover
  - Edge cases: empty secret, 1-byte secret, 20-byte TOTP secret

**Recommendations:**
- Implement rigorous test suite BEFORE integration with MPC nodes
- Add property-based tests (e.g., for all valid 2-share combinations)
- Verify GF(256) log/exp tables against reference implementation
- Consider security audit before production deployment
- Document the Lagrange interpolation formula explicitly in code

---

### Critical: TOTP Secret Lifecycle & Memory Safety

**Risk:** TOTP secret must never persist to storage; only transient in-memory handling with explicit zeroing

**Files Affected:** `twofa/internal/services/twofaService/` (setup.go, verify.go, disable.go), `twofa/internal/services/twofaService/totp/totp.go`

**Problems:**
- No built-in memory zeroization in Go — developers must manually clear sensitive bytes
- Garbage collection timing non-deterministic; secrets could remain in heap
- If any logging accidentally includes secret (despite rules), production exposure
- Shamir recombination step produces secret in memory; must be cleared immediately after TOTP validation

**Current Mitigation:**
- Specification forbids persistent storage
- Specification requires `zeroize` after split and after validation
- Design specifies memory-only handling

**Recommendations:**
- Implement explicit zeroization helper:
  ```go
  func zeroize(data []byte) {
      for i := range data {
          data[i] = 0
      }
  }
  ```
- Use in service code immediately after use: `defer zeroize(secret)`
- Add panic if secret accidentally logged (check in logging middleware)
- Document that Go's runtime makes stronger guarantees than some languages but not perfect
- Consider stack allocation for small secrets (stack clearing happens on function exit)

---

### High: JWT Key Management & Distribution

**Risk:** RS256 requires private key in Auth Service and public key distribution to Gateway/TwoFA/MPC services

**Files:** `auth/internal/services/authService/jwt.go` (generation), `auth/cmd/app/main.go` (key loading), `gateway/config/config.go` (public key loading), all other services

**Problems:**
- Private key must be loaded from config or environment at startup — **never expose in logs**
- Public key distribution mechanism not yet specified — hard-coded, environment var, or discovery?
- Key rotation strategy undefined — old tokens become invalid
- If private key compromised, all issued tokens become forgeable

**Current Mitigation:**
- Specification marks "НИКОГДА не логировать секретные данные"
- Design specifies RS256 for asymmetric verification

**Recommendations:**
- Implement secure key loading:
  - Load RSA keys from file system (not inline in code)
  - File permissions: 0600 for private key
  - Check and log if key files are world-readable (security warning)
- For public key distribution:
  - Option 1: Hard-coded in services (requires code deploy to rotate)
  - Option 2: Fetch from Auth Service on startup (adds dependency, latency)
  - Option 3: Environment variable (simpler, fits config model)
  - Recommend Option 3 with clear documentation
- Implement key rotation strategy:
  - Support multiple public keys for grace period during rotation
  - Add `kid` (key ID) header to JWT for versioning
- Add metrics: `auth_jwt_generation_total`, `auth_jwt_validation_errors_total` (track validation failures to detect compromise early)

---

### High: AES-256-GCM Nonce Management in MPC Nodes

**Risk:** Each share encryption uses AES-256-GCM; nonce must be unique per operation

**Files:** `mpc/internal/crypto/aes.go`, `mpc/internal/services/shareService/store.go`

**Problems:**
- Nonce generated via `crypto/rand` — must be unique across all encryptions with same key
- If same nonce reused with same key, GCM security breaks (ciphertext allows plaintext recovery)
- Nonce stored with ciphertext in database; must be at least 12 bytes (Go crypto/cipher/gcm default)
- Specification requires storing nonce with ciphertext for decryption, but "at-rest" implies key is persistent

**Current Mitigation:**
- Specification says "nonce через crypto/rand" and database schema includes `nonce BYTEA`
- Design implies nonce never reused (generated fresh each time)

**Recommendations:**
- Implement nonce counter as additional safety:
  - Store operation counter per key in database (with increment)
  - Use counter as additional security layer (optional, performance trade-off)
- Document nonce uniqueness guarantee:
  - Crypto/rand is cryptographically secure
  - In practice, 96-bit nonce collision vanishingly rare
  - Still, add unit tests validating uniqueness across multiple encryptions
- Audit database schema to ensure nonce is stored for every share:
  ```sql
  CREATE TABLE shares (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    share_index INT NOT NULL,
    encrypted_data BYTEA NOT NULL,
    nonce BYTEA NOT NULL,  -- MUST be present
    created_at TIMESTAMP NOT NULL,
    UNIQUE(user_id, share_index)
  );
  ```
- Decrypt operation MUST validate nonce length (12 bytes expected)

---

### Medium: Rate Limiting Implementation

**Risk:** 2FA verification limited to 5 attempts per 5 minutes per user_id

**Files:** `twofa/internal/services/twofaService/verify.go`, potentially `gateway/` for centralized rate limiting

**Problems:**
- No specification of where rate limiting is enforced (TwoFA Service or Gateway?)
- Redis integration exists in other services but rate limiting approach undefined
- No protection against distributed attacks (attacker from multiple IP addresses)
- No mechanism to unblock false positive lockouts (admin override)

**Current Mitigation:**
- Specification states requirement clearly
- Redis available for distributed state

**Recommendations:**
- Implement in TwoFA Service using Redis key: `2fa_verify_attempts:{user_id}`
- Use INCR with expiry: increment counter, set TTL to 5 minutes if new key
- Return gRPC status `ResourceExhausted` when limit reached
- Add Prometheus metric: `twofa_rate_limit_exceeded_total`
- Document: this prevents brute force on same user_id but not distributed attacks
- Consider adding optional admin override mechanism (flag in service to disable limit)

---

## Architecture & Design Concerns

### High: Cross-Service Communication Resilience

**Risk:** TwoFA Service must contact 3 MPC nodes in parallel; one failure fails entire operation

**Files:** `twofa/internal/services/twofaService/setup.go`, `twofa/internal/clients/mpc/client.go`

**Problems:**
- Specification: "Если хотя бы одна нода недоступна — ошибка"
- This means Setup2FA requires all 3 MPC nodes healthy — tight coupling
- No retry mechanism specified
- No circuit breaker for failing nodes
- Timeout set to 5 seconds per call; no adaptive backoff

**Current Mitigation:**
- Design specifies 5-second timeout and parallel calls
- Specification explicitly requires all 3 nodes available

**Recommendations:**
- Implement retry logic with exponential backoff:
  ```
  Attempt 1: immediate
  Attempt 2: 100ms delay
  Attempt 3: 200ms delay
  Max total time: 5 seconds
  ```
- Add circuit breaker pattern for nodes (track consecutive failures)
- Document that system cannot operate with < 3 healthy MPC nodes
- Add metrics to monitor node health:
  - `twofa_mpc_node_availability` (per node)
  - `twofa_mpc_share_operation_failures_total` (operation, node, reason)
- Recommendation: Consider fallback for future resilience (not in current spec, but design for it):
  - Could use 3-of-5 instead of 2-of-3 to tolerate 2 node failures
  - Would add complexity; only consider if high availability required

---

### High: Error Handling Consistency Across Services

**Risk:** Four services with different error contexts; no unified error handling pattern yet

**Files:** All `internal/api/*_service_api/*.go` handler files (not yet created)

**Problems:**
- Specification requires gRPC status codes: `InvalidArgument, NotFound, Unauthenticated, AlreadyExists, Internal`
- No documented mapping from business logic errors to gRPC statuses
- No error context/tracing (e.g., request ID propagation)
- Duplicate error handling code across services

**Current Mitigation:**
- Specification defines allowed status codes

**Recommendations:**
- Create shared error mapping interface (e.g., in `pkg/errors/` if shared) or document per-service:
  - `InvalidArgument`: validation failure (password policy, OTP invalid)
  - `NotFound`: resource doesn't exist (user, session, 2FA record)
  - `Unauthenticated`: token invalid/expired (ValidateToken failure)
  - `AlreadyExists`: duplicate email, 2FA already enabled
  - `Internal`: database error, MPC node failure, crypto error
- Implement request ID logging:
  - Add `request_id` to gRPC metadata from Gateway
  - Include in all logs for tracing
  - Return in gRPC response metadata
- Document error codes and messages in service documentation

---

### Medium: Database Initialization & Schema Consistency

**Risk:** Four services use PostgreSQL; schema initialization via code-based `initTables()`

**Files:** `auth/internal/storage/pgstorage/pgstorage.go`, `twofa/internal/storage/pgstorage/pgstorage.go`, `mpc/internal/storage/pgstorage/pgstorage.go`

**Problems:**
- No migration framework specified (Flyway, golang-migrate, or raw SQL)
- Schema changes require code restart
- Multiple services may try to initialize tables concurrently
- No version tracking of schema
- Testing requires seeding test data

**Current Mitigation:**
- Design specifies `initTables` pattern
- Each service has its own storage layer

**Recommendations:**
- Implement idempotent initialization:
  ```go
  // Use CREATE TABLE IF NOT EXISTS
  // Check current schema version before migrations
  ```
- Add schema version table:
  ```sql
  CREATE TABLE IF NOT EXISTS schema_version (
    service VARCHAR(50) PRIMARY KEY,
    version INT NOT NULL,
    applied_at TIMESTAMP NOT NULL
  );
  ```
- Document expected schema for each service in service documentation (`workspace/02 - Services/`)
- Create separate database user per service (auth, twofa, mpc) with minimal permissions
- Test with concurrent service startup to catch race conditions

---

## Testing & Quality Concerns

### High: Incomplete Test Coverage Specification

**Risk:** Testing requirements documented but scope undefined

**Files:** All service test files (not yet created), `auth/internal/services/authService/password_validation_test.go`, `twofa/internal/services/twofaService/shamir/shamir_test.go`

**Problems:**
- Specification requires tests for: password validation, Shamir split/combine, TOTP, AES-256-GCM
- No integration test framework specified (testcontainers? Docker Compose?)
- No e2e test specification (Frontend through all services)
- Test coverage targets not defined
- Mocking strategy for external services (Redis, Kafka, PostgreSQL) not documented

**Current Mitigation:**
- TODO.md explicitly lists test tasks

**Recommendations:**
- Define minimum coverage targets:
  - `auth`: 90% for password validation, JWT generation; 80% overall
  - `twofa`: 95% for Shamir (critical), 90% for TOTP, 80% overall
  - `mpc`: 95% for AES-256-GCM, 80% overall
  - `gateway`: 70% (less critical, mostly routing)
- Implement unit tests using Go's standard testing package:
  - Create `*_test.go` files in same package as code
  - Use interfaces for dependency injection in tests
- Integration tests:
  - Use `testcontainers-go` for PostgreSQL, Redis, Kafka
  - Or `docker-compose.test.yaml` with test script
- E2E tests:
  - Separate test suite using gRPC client libraries
  - May require dedicated test environment

---

### Medium: Secrets in Test Fixtures & Logs

**Risk:** Test data may accidentally include realistic credentials

**Files:** All test files (e.g., password_validation_test.go, shamir_test.go)

**Problems:**
- Tests for password validation need valid/invalid examples
- Tests for Shamir need actual secret bytes
- Test output (logs, error messages) must not expose secrets
- CI/CD logs may be world-readable

**Current Mitigation:**
- Specification forbids logging secrets

**Recommendations:**
- For password tests: use clearly dummy passwords (`ValidPassword1!`, `Invalid`, etc.)
- For cryptographic tests: use fixed test vectors (not random secrets)
  - Example: `secret := []byte{0x01, 0x02, ...}` with documentation
- Review all test files before commit:
  - Search for actual email addresses, tokens, keys
  - Use constants for repeated test data
- Configure test output verbosity:
  - Run tests with `-v` flag only in CI for debugging
  - Default test runs suppress sensitive output

---

## Infrastructure & Deployment Concerns

### High: Configuration Management Across Services

**Risk:** Four independent services with separate `config.yaml` files; no centralized configuration

**Files:** `auth/config.yaml`, `twofa/config.yaml`, `mpc/config.yaml`, `gateway/config.yaml`

**Problems:**
- Each service has its own configuration file
- Cross-service settings (e.g., MPC node addresses in TwoFA config) must be kept in sync
- Environment-specific configurations (dev, staging, prod) not addressed
- No validation of configuration completeness at startup
- Secrets (database passwords, encryption keys) unclear whether in config or environment

**Current Mitigation:**
- Design specifies `config.yaml` with `gopkg.in/yaml.v3`
- Specification mentions `ENCRYPTION_KEY` as environment variable

**Recommendations:**
- Separate secrets from configuration:
  - `config.yaml`: non-sensitive settings (service addresses, timeouts, rate limits)
  - Environment variables: secrets (database password, Redis password, encryption keys, JWT keys)
- Implement configuration validation at startup:
  ```go
  func (c *Config) Validate() error {
      if c.PostgreSQL.DSN == "" {
          return fmt.Errorf("missing POSTGRES_DSN")
      }
      // ... validate all required fields
  }
  ```
- Document required environment variables per service (e.g., in README or `.env.example`)
- For MPC node addresses in TwoFA config:
  - Consider environment variable override: `TWOFA_MPC_NODES=node1:5001,node2:5002,node3:5003`
  - Or service discovery mechanism (future enhancement)

---

### Medium: Kafka Reliability & Message Ordering

**Risk:** Kafka used for audit events but reliability strategy undefined

**Files:** `auth/internal/bootstrap/kafka_producer.go`, `twofa/internal/bootstrap/kafka_producer.go`, `mpc/internal/bootstrap/kafka_producer.go`

**Problems:**
- No specification of Kafka topic names or schemas
- Producer configuration not defined (acks, retries, batching)
- No consumer implementation for audit log processing (generates events but who reads them?)
- Message ordering across services not guaranteed
- Failed sends not handled (fire-and-forget? retry?)

**Current Mitigation:**
- Design includes Kafka integration
- Services produce events: `user.registered`, `user.logged_in`, `token.refreshed`, `2fa.verified`, etc.

**Recommendations:**
- Define Kafka topics and schemas:
  - Topic: `auth-events` (Partitions: 3 for parallelism)
  - Topic: `twofa-events`
  - Topic: `mpc-events`
  - Topic: `audit-log` (aggregated)
- Document event schema (JSON):
  ```json
  {
    "user_id": "uuid",
    "operation": "2fa.verified",
    "timestamp": "2026-04-11T12:34:56Z",
    "node_id": "mpc-1"
  }
  ```
- Implement producer with reliability:
  ```go
  writer := kafka.NewWriter(kafka.WriterConfig{
      Brokers: brokerAddrs,
      Topic: topic,
      Compression: kafka.Gzip,
      MaxAttempts: 3,
      WriteBackoffMin: 100 * time.Millisecond,
      WriteBackoffMax: 1 * time.Second,
  })
  ```
- Add error handling:
  - Log failed sends as errors (don't panic)
  - Metric: `kafka_produce_failures_total`
- Document that audit events are best-effort (not transactional with database operations)

---

### Medium: Redis Session & Rate Limit Management

**Risk:** Redis used for refresh tokens and rate limiting; no persistence strategy

**Files:** `auth/internal/storage/redisstorage/session.go`, `twofa/` (rate limiting)

**Problems:**
- Redis is in-memory; restart loses all sessions
- Refresh tokens become invalid if Redis cluster goes down
- No backup/persistence strategy specified (RDB snapshots, AOF)
- Rate limit state lost on restart (minor risk, resets limit)
- No eviction policy specified (what happens when Redis runs out of memory?)

**Current Mitigation:**
- Design accepts Redis for temporary state
- TTL specified: refresh tokens 7 days, rate limits 5 minutes

**Recommendations:**
- Document Redis deployment expectations:
  - Persistence: enable AOF (append-only file) for production
  - Memory limit with eviction policy: `allkeys-lru` or `volatile-lru`
  - Replication: use Redis Sentinel or Cluster for HA
- Implement graceful degradation:
  - If Redis unavailable for rate limiting: log warning, allow request (degrade to no limiting)
  - If Redis unavailable for refresh tokens: return 503 Service Unavailable
- Metrics to monitor:
  - `redis_connection_errors_total`
  - `redis_operation_duration_seconds` (histogram by operation)

---

## Specification & Documentation Concerns

### Medium: Incomplete Gateway Specification

**Risk:** API Gateway specified but implementation details minimal

**Files:** `gateway/` (entire service)

**Problems:**
- Gateway exists in directory structure but no proto definitions or implementation details
- REST→gRPC translation not specified (request/response mapping)
- Rate limiting placement unclear: Gateway or individual services?
- Authentication enforcement (ValidateToken) not detailed
- CORS, request validation, response formatting not documented
- Error response format not standardized

**Current Mitigation:**
- Architecture diagram shows Gateway as entry point
- CLAUDE.md mentions "REST→gRPC, rate limiting"

**Recommendations:**
- Create API specification document (`workspace/02 - Services/API Gateway.md`):
  - Define REST endpoint mappings (e.g., `POST /api/auth/register` → `AuthService.Register`)
  - Document request/response formats
  - Specify authentication flow (Authorization header with access token)
- Implement Gateway patterns:
  - Use gRPC protocol buffers for service definitions
  - Add reverse proxy middleware (e.g., grpc-gateway for REST→gRPC)
  - Centralized request validation & rate limiting at Gateway
  - Consistent error response format (JSON):
    ```json
    {
      "error": "INVALID_ARGUMENT",
      "message": "Email already registered",
      "request_id": "uuid"
    }
    ```

---

### Medium: TOTP Provisioning & QR Code Generation

**Risk:** Setup2FA returns provisioning URI but client-side implementation undefined

**Files:** `twofa/internal/services/twofaService/totp/totp.go`

**Problems:**
- Specification requires "provisioning URI (otpauth://totp/...)"
- QR code generation not mentioned (frontend responsibility?)
- What if user loses their authenticator app before verifying 2FA?
- Backup codes specification vague ("generate 10 backup-codes, hash each")
- Backup code usage/rotation not documented

**Current Mitigation:**
- TOTP RFC 6238 documentation created (`workspace/03 - Security/TOTP RFC 6238.md`)
- 10 backup codes generated in Setup2FA

**Recommendations:**
- Document provisioning URI generation:
  - Format: `otpauth://totp/user%40example.com?secret=JBSWY3DPEBLW64TMMQ======&issuer=MPC2FA`
  - Library: `github.com/pquerna/otp` can generate this
  - Frontend: generate QR code from URI (client-side library: `qrcode.js`)
- Handle Setup2FA cancellation:
  - If user doesn't verify within 24 hours, delete shares and metadata
  - Implement background job: `SELECT * FROM twofa_records WHERE is_enabled=false AND created_at < now() - interval '24 hours'`
  - Delete shares from MPC nodes and record from database
- Document backup codes:
  - Format: random 8-character alphanumeric (generated at setup)
  - Hashed with bcrypt (same as password) before storage
  - One-time use: increment `used_count`, disable when >= 10
  - Store `created_at, used_at, used_count` for audit
  - User must save during setup (shown once, not retrieved later)

---

## Missing Features & Future Risks

### Medium: No Admin/Support Operations Defined

**Risk:** Production systems require administrative capabilities; not in spec

**Problems:**
- What if user loses access to authenticator app?
- How does support reset/disable 2FA for a user?
- No admin console or API endpoints specified
- Audit log is write-only; no query interface

**Recommendations:**
- Define admin operations (for future phases):
  - `AdminDisable2FA(user_id)` — force-disable without verification
  - `AdminResetPassword(user_id)` — if email recovery needed
  - `AuditLog.Query(user_id, operation, date_range)` — read audit events
- Require strong authentication for admin endpoints (e.g., different API key)
- Log all admin operations with separate audit trail

---

### Medium: No Account Recovery Mechanism

**Risk:** If user forgets password AND loses authenticator, account is permanently locked

**Problems:**
- Specification does not include email verification or password reset
- 2FA cannot be disabled without valid OTP
- No recovery codes beyond 10 backup codes (all may be used)
- No way to regain access if locked out

**Current Mitigation:**
- Backup codes allow recovery if authenticator lost
- Specification explicitly forbids "email verification, OAuth, SSO"

**Recommendations:**
- For scope creep: consider in future
- Current design acceptable for PoC/thesis but document limitation
- Recommend: implement phone-based recovery or trusted device tokens in future phases

---

### Low: Monitoring & Alerting Not Specified

**Risk:** Prometheus configured in stack but dashboards/alerts undefined

**Problems:**
- No alert thresholds specified
- No runbook for common failures
- No SLO/SLA targets

**Current Mitigation:**
- Prometheus + Grafana available
- Specification requires metrics in each service

**Recommendations:**
- Define critical alerts:
  - MPC node down (affects 2FA setup/verify)
  - PostgreSQL connection failures
  - Redis unavailable
  - Kafka producer failures
  - Auth token generation failures
- Create Grafana dashboard:
  - Service health (request rate, error rate, latency)
  - Resource usage (CPU, memory, connections)
  - Business metrics (registrations/day, 2FA setups, failures)

---

## Code Quality Observations

### Medium: No Linting/Formatting Standards Defined

**Problems:**
- No `.eslintrc`, `.prettierrc`, or Go linting configuration specified
- Import organization not documented
- Naming conventions not explicit (camelCase, snake_case?)

**Recommendations:**
- Add `.golangci.yml` for linter configuration:
  ```yaml
  linters:
    enable:
      - errcheck
      - govet
      - ineffassign
      - unused
      - staticcheck
  ```
- Define Go code style:
  - Use `gofmt` standard (enforced)
  - Use `go vet` for common errors
  - Add pre-commit hooks in `.git/hooks/pre-commit`
- Document naming conventions in CONVENTIONS.md

---

## Summary Table

| Concern | Severity | Status | File(s) |
|---------|----------|--------|---------|
| Shamir implementation correctness | Critical | Not started | `twofa/internal/services/twofaService/shamir/` |
| TOTP secret memory safety | Critical | Not started | `twofa/internal/services/twofaService/` |
| JWT key management | High | Not started | `auth/internal/services/authService/jwt.go` |
| AES-256-GCM nonce management | High | Not started | `mpc/internal/crypto/aes.go` |
| MPC node resilience | High | Not started | `twofa/internal/services/twofaService/setup.go` |
| Error handling consistency | High | Not started | All service handlers |
| Rate limiting implementation | Medium | Not started | `twofa/internal/services/twofaService/verify.go` |
| Database schema initialization | Medium | Not started | All `pgstorage` packages |
| Test coverage specification | High | Not started | All test files |
| Configuration management | High | Not started | All `config.yaml` files |
| Kafka reliability | Medium | Not started | All kafka producer files |
| Redis persistence | Medium | Not started | `auth/internal/storage/redisstorage/` |
| Gateway implementation | Medium | Not started | `gateway/` |
| TOTP provisioning & backup codes | Medium | Not started | `twofa/internal/services/twofaService/totp/` |
| Admin operations | Medium | Future phase | Not applicable |
| Account recovery | Medium | Out of scope | Not applicable |
| Monitoring & alerting | Low | Not started | `monitoring/` |

---

*Concerns audit: 2026-04-11*
