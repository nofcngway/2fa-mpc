# Architecture

**Analysis Date:** 2026-04-11

## Pattern Overview

**Overall:** Clean Architecture microservices with gRPC inter-service communication

**Key Characteristics:**
- Four independent Go services, each following Clean Architecture pattern (handler → service → repository)
- gRPC for all inter-service communication (not HTTP)
- REST/HTTP only at API Gateway boundary (REST ↔ HTTP → gRPC translation)
- Dependency injection through bootstrap factories for testability and loose coupling
- Layered architecture: api/ (proto) → cmd/app/ (entry) → config/ → internal/{api, bootstrap, services, storage, models, pb, middleware}

## Layers

**API Layer (gRPC):**
- Purpose: Handle incoming gRPC requests and outgoing gRPC calls to other services
- Location: `<service>/internal/api/<service>_service_api/`
- Contains: gRPC handler files (one file per RPC method), handler registration
- Depends on: Domain services (business logic), models
- Used by: gRPC clients in other services and gateway

**Service Layer (Business Logic):**
- Purpose: Implement domain logic, orchestration, and security-critical operations
- Location: `<service>/internal/services/<serviceName>/`
- Contains: Business logic files (one file per operation), interfaces for dependencies
- Depends on: Storage repositories, external clients (e.g., MPC, Redis), models
- Used by: API handlers, other service layers

**Storage Layer (Data Access):**
- Purpose: Abstract database and external data stores with repository pattern
- Location: `<service>/internal/storage/<storage-type>/`
- Contains: PostgreSQL repository (`pgstorage/`), Redis session storage (`redisstorage/`), MPC clients (`clients/`)
- Depends on: Models, external clients (pgx, redis-go)
- Used by: Service layer

**Configuration & Bootstrap:**
- Purpose: Initialize dependencies and wire components together
- Location: `<service>/config/config.go`, `<service>/internal/bootstrap/`
- Contains: Config loading from YAML, DI factories for each dependency (database, cache, gRPC clients, producers)
- Depends on: External libraries (pgx, redis, grpc, kafka)
- Used by: main.go

**Middleware Layer:**
- Purpose: Cross-cutting concerns (logging, metrics, error handling, authentication)
- Location: `<service>/internal/middleware/interceptors.go`
- Contains: gRPC interceptors for request/response logging, Prometheus metrics, recovery from panics
- Used by: gRPC server registration

## Data Flow

**User Registration:**

1. Frontend → Gateway (HTTP POST /register)
2. Gateway handler → Auth Service (gRPC Register)
3. Auth handler → AuthService.Register() (business logic)
4. AuthService → PGStorage.CreateUser() (validate, bcrypt hash, INSERT)
5. AuthService → KafkaProducer.Publish("user.registered")
6. Gateway → Frontend (HTTP response)

**2FA Setup (TOTP Secret Distribution):**

1. Frontend → Gateway (HTTP POST /2fa/setup)
2. Gateway validates token via Auth.ValidateToken()
3. Gateway → TwoFA Service (gRPC Setup2FA)
4. TwoFA handler → TwoFAService.Setup()
5. TwoFAService:
   - Generates 20-byte TOTP secret
   - Calls shamir.Split(secret, n=3, threshold=2) → [share1, share2, share3]
   - Zeroizes secret from memory
   - Calls MPC clients (3 parallel): StoreShare(user_id, index, share)
6. MPC Node handler → ShareService.Store()
7. ShareService → Encryptor.Encrypt(share) (AES-256-GCM)
8. PGStorage.CreateShare(user_id, index, encrypted_data, nonce)
9. MPC → KafkaProducer.Publish("share.stored")
10. TwoFA → PostgreSQL: INSERT 2fa_records (is_enabled=false)
11. TwoFA → KafkaProducer.Publish("2fa.setup")
12. Gateway → Frontend (HTTP response with provisioning URI)

**2FA Verification (Secret Reconstruction):**

1. Frontend → Gateway (HTTP POST /2fa/verify)
2. Gateway → TwoFA Service (gRPC Verify2FA)
3. TwoFA handler → TwoFAService.Verify()
4. Check rate limit: RedisStorage.CheckAttempts()
5. TwoFAService:
   - Calls MPC clients (2 of 3): RetrieveShare(user_id, index)
   - MPC node → PGStorage.GetShare() → Decryptor.Decrypt(encrypted_data, nonce) → share
   - Calls shamir.Combine([share1, share2]) → secret
   - Calls totp.Validate(secret, otp_code, ±1_window)
   - Zeroizes secret
6. TwoFA → PostgreSQL: UPDATE 2fa_records (is_enabled=true if first verify)
7. TwoFA → KafkaProducer.Publish("2fa.verified")
8. Gateway → Frontend (HTTP response)

**State Management:**

- **Session State**: Refresh tokens stored in Redis with 7-day TTL (Auth Service)
- **Rate Limit Counters**: Redis counters per user_id, 5-minute TTL (Gateway)
- **Persistent State**: PostgreSQL stores users, sessions, 2FA metadata, encrypted shares, audit logs
- **Transient Secrets**: TOTP secrets and reconstructed shares only exist in memory during operation, then zeroized

## Key Abstractions

**Shamir Secret Sharing:**
- Purpose: Distribute TOTP secret across 3 MPC nodes with 2-of-3 threshold
- Examples: `twofa/internal/services/twofaService/shamir/shamir.go`, `gf256.go`
- Pattern: Polynomial-based in GF(256), Lagrange interpolation for reconstruction. Never store full secret.

**TOTP Generation & Validation:**
- Purpose: Generate provisioning URI for QR code and validate OTP codes with ±1 time window
- Examples: `twofa/internal/services/twofaService/totp/totp.go`
- Pattern: RFC 6238, crypto/rand for seed, time-based counter

**JWT Token Management:**
- Purpose: Issue RS256-signed access/refresh tokens with distinct TTLs
- Examples: `auth/internal/services/authService/jwt.go`
- Pattern: RSA asymmetric signing (private key in Auth Service only), public key distributed to Gateway/TwoFA for verification

**Password Validation:**
- Purpose: Enforce security policy on user passwords
- Examples: `auth/internal/services/authService/password_validation.go`
- Pattern: Rule-based validation (length, character classes, no keyboard sequences)

**AES-256-GCM Encryption:**
- Purpose: Encrypt shares at-rest in MPC node PostgreSQL
- Examples: `mpc/internal/crypto/aes.go`
- Pattern: Each operation gets unique nonce (crypto/rand, 12 bytes), ciphertext + nonce stored together

**Storage Repository Pattern:**
- Purpose: Abstract database access with dependency injection
- Examples: `auth/internal/storage/pgstorage/user.go`, `mpc/internal/storage/pgstorage/share.go`
- Pattern: Interfaces for dependencies (CRUD operations), pgx client injected at construction

**gRPC Clients:**
- Purpose: Call other services with timeout and error handling
- Examples: `twofa/internal/clients/mpc/client.go`
- Pattern: Lazy initialization in bootstrap, context with timeout (5s for MPC calls), status code mapping

## Entry Points

**Auth Service:**
- Location: `auth/cmd/app/main.go`
- Triggers: Binary execution
- Responsibilities: Load config → bootstrap dependencies → register gRPC handlers → start server → graceful shutdown

**TwoFA Service:**
- Location: `twofa/cmd/app/main.go`
- Triggers: Binary execution
- Responsibilities: Load config → bootstrap MPC clients (3 connections) → register handlers → start server

**MPC Node Service:**
- Location: `mpc/cmd/app/main.go`
- Triggers: Binary execution (one per node, NODE_ID from config)
- Responsibilities: Load config → set up encryption key → bootstrap database → start gRPC server

**API Gateway:**
- Location: `gateway/cmd/app/main.go`
- Triggers: Binary execution
- Responsibilities: Register HTTP handlers → create gRPC clients to Auth/TwoFA → start HTTP server → graceful shutdown

## Error Handling

**Strategy:** gRPC status codes for service-to-service communication, HTTP status codes for client responses

**Patterns:**
- InvalidArgument: Client sends malformed request (e.g., invalid email, weak password)
- NotFound: Resource doesn't exist (user, 2FA record)
- Unauthenticated: JWT invalid or user not authenticated
- AlreadyExists: Email already registered, 2FA already enabled
- PermissionDenied: User not authorized for operation
- Internal: Server-side error (database connection, encryption failure)
- All errors logged with slog structured logging (context preserved, secrets never logged)

## Cross-Cutting Concerns

**Logging:** 
- Framework: Go standard library `log/slog` (structured logging)
- Approach: Middleware interceptors add request ID, timestamp, method name. Service layer logs state transitions. Never log secrets, passwords, shares, TOTP, encryption keys.

**Validation:** 
- Approach: Handler validates input type/format → Service validates business logic (password rules, TOTP format). Return InvalidArgument on failure.

**Authentication:** 
- Approach: Gateway validates JWT via Auth.ValidateToken() → extracts user_id → passes to downstream services in context. TwoFA/MPC services check user_id in context or request metadata.

**Monitoring:**
- Metrics: Prometheus counters (requests_total, errors_total) and histograms (request_duration_seconds) per service and method
- Health check: gRPC Health Check Protocol endpoint on each service
- Graceful shutdown: Signal handlers close database connections, drain in-flight requests (10s timeout)

---

*Architecture analysis: 2026-04-11*
