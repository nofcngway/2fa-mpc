# Architecture

**Analysis Date:** 2026-04-11

## Pattern Overview

**Overall:** Microservices with Clean Architecture (handler → service → repository pattern)

**Key Characteristics:**
- API Gateway as single HTTP entry point, all inter-service communication via gRPC
- Distributed secret storage using Shamir Secret Sharing (2-of-3 threshold)
- Event-driven audit logging through Kafka
- Clear separation of concerns: Authentication, 2FA orchestration, MPC node storage
- Dependency Injection via bootstrap factories

## Layers

**API Layer (gRPC Handlers):**
- Purpose: Handle incoming gRPC RPC calls, validate requests, coordinate with service layer
- Location: `<service>/internal/api/<service>_service_api/`
- Contains: One file per RPC method (e.g., `register.go`, `login.go`, `setup.go`)
- Depends on: Service layer (business logic), protobuf models
- Used by: gRPC clients (Gateway, other services)

**Service Layer (Business Logic):**
- Purpose: Implement domain logic, orchestrate operations, manage state transitions
- Location: `<service>/internal/services/<serviceName>/`
- Contains: One file per major operation (e.g., `register.go`, `login.go`, `password_validation.go`)
- Depends on: Repository layer (data access), external services (Redis, Kafka)
- Used by: API layer (handlers)

**Storage Layer (Data Access):**
- Purpose: Abstract database and cache access, provide persistence
- Location: `<service>/internal/storage/pgstorage/` and `<service>/internal/storage/redisstorage/`
- Contains: PostgreSQL repositories using pgx directly (no ORM), Redis session management
- Depends on: Database drivers (pgx v5, redis/go-redis)
- Used by: Service layer

**Bootstrap Layer (Dependency Injection):**
- Purpose: Create and wire all service dependencies, manage component lifecycle
- Location: `<service>/internal/bootstrap/`
- Contains: Factory functions for each major component (services, storage, clients, server)
- Depends on: All other layers
- Used by: `main()` in `cmd/app/main.go`

**Middleware Layer:**
- Purpose: Cross-cutting concerns: authentication, rate limiting, logging, metrics, error recovery
- Location: `<service>/internal/middleware/interceptors.go`
- Contains: gRPC unary and stream interceptors
- Depends on: Service layer (for validation), logging (slog), metrics (Prometheus)
- Used by: gRPC server configuration in bootstrap

## Data Flow

**User Registration:**

1. Frontend → Gateway: `POST /register {email, password}` (HTTPS)
2. Gateway → Auth Service: gRPC `Register(email, password)` call
3. Auth Service:
   - Validates password (length, character classes, no sequences)
   - Hashes password with bcrypt (cost=12)
   - Queries PostgreSQL to check email uniqueness
   - Inserts user record into `users` table
   - Publishes `user.registered` event to Kafka
4. Auth Service → Gateway → Frontend: Success response

**User Login:**

1. Frontend → Gateway: `POST /login {email, password}` (HTTPS)
2. Gateway → Auth Service: gRPC `Login(email, password)` call
3. Auth Service:
   - Queries PostgreSQL for user by email
   - Verifies password hash with bcrypt
   - Generates JWT pair: access token (15 min) + refresh token (7 days)
   - Stores refresh token in Redis with TTL=7 days
   - Publishes `user.logged_in` event to Kafka
4. Auth Service → Gateway → Frontend: `{access_token, refresh_token}`

**Setup 2FA (Distributed Secret Storage):**

1. Frontend → Gateway: `POST /2fa/setup` with access token (HTTPS)
2. Gateway validates token via Auth.ValidateToken gRPC call
3. Gateway → TwoFA Service: gRPC `Setup2FA(user_id)` call
4. TwoFA Service:
   - Generates TOTP secret (20 random bytes)
   - Splits secret using Shamir Secret Sharing (3 shares, threshold 2)
   - **Zeroizes TOTP secret from memory**
   - Sends each share to respective MPC node via gRPC:
     - `MPC-1.StoreShare(user_id, index=1, share1)`
     - `MPC-2.StoreShare(user_id, index=2, share2)`
     - `MPC-3.StoreShare(user_id, index=3, share3)`
5. Each MPC Node:
   - Receives share
   - Encrypts with AES-256-GCM (at-rest encryption)
   - Stores encrypted data + nonce in PostgreSQL
6. TwoFA Service:
   - Inserts `2fa_records` metadata into PostgreSQL (is_enabled=false)
   - Publishes `2fa.setup` event to Kafka
7. TwoFA Service → Gateway → Frontend: Provisioning URI (for QR code)

**Verify 2FA (Reconstruct Secret from Shares):**

1. Frontend → Gateway: `POST /2fa/verify {otp_code}` (HTTPS)
2. Gateway → TwoFA Service: gRPC `Verify2FA(user_id, otp_code)` call
3. TwoFA Service:
   - Checks rate limit (max 5 attempts per 5 minutes)
   - Requests shares from 2 MPC nodes (gRPC calls):
     - `MPC-1.RetrieveShare(user_id, index=1)`
     - `MPC-2.RetrieveShare(user_id, index=2)`
4. Each MPC Node:
   - Queries PostgreSQL for encrypted share
   - Decrypts using AES-256-GCM
   - Returns decrypted share bytes
5. TwoFA Service:
   - Uses Shamir Combine to reconstruct TOTP secret from 2 shares
   - Validates OTP code against secret (RFC 6238, ±1 time window)
   - **Zeroizes TOTP secret from memory**
   - Updates `2fa_records` (is_enabled=true if first verification)
   - Publishes `2fa.verified` event to Kafka
6. TwoFA Service → Gateway → Frontend: Success

**Disable 2FA:**

1. Frontend → Gateway: `POST /2fa/disable {otp_code}` (HTTPS)
2. TwoFA Service:
   - Verifies OTP (same flow as Verify2FA, steps 3-5)
   - If valid, requests deletion from all MPC nodes:
     - `MPC-1.DeleteShare(user_id)`
     - `MPC-2.DeleteShare(user_id)`
     - `MPC-3.DeleteShare(user_id)`
3. Each MPC Node:
   - Deletes all share records for user from PostgreSQL
4. TwoFA Service:
   - Deletes `2fa_records` from PostgreSQL
   - Publishes `2fa.disabled` event to Kafka
5. TwoFA Service → Gateway → Frontend: Success

**State Management:**

- **JWT Tokens**: Generated by Auth Service (RS256), stored client-side, validated on each gRPC call
- **Refresh Tokens**: Stored in Redis with TTL, used to rotate JWT pairs
- **TOTP Secrets**: Never persisted in plaintext — reconstructed on-demand from encrypted shares
- **Shares**: Stored encrypted at-rest in MPC node databases, retrieved only when needed for verification
- **Rate Limits**: Counters stored in Redis (Gateway for general, TwoFA for 2FA attempts)
- **Audit Events**: Immutable log in PostgreSQL + real-time stream to Kafka

## Key Abstractions

**AuthService:**
- Purpose: Encapsulates user authentication and JWT management
- Examples: `auth/internal/services/authService/auth_service.go`, `register.go`, `login.go`, `jwt.go`
- Pattern: Dependency-injected service with methods for each operation (Register, Login, RefreshToken, Logout, ValidateToken)

**TwoFAService:**
- Purpose: Orchestrates 2FA setup, verification, and disable operations with MPC node coordination
- Examples: `twofa/internal/services/twofaService/twofa_service.go`, `setup.go`, `verify.go`
- Pattern: Coordinates multiple gRPC calls to MPC nodes, manages Shamir split/combine lifecycle

**ShamirSecretSharing:**
- Purpose: Splits and reconstructs secrets using Shamir Secret Sharing (2-of-3)
- Examples: `twofa/internal/services/twofaService/shamir/shamir.go`
- Pattern: Custom implementation in GF(256) with Lagrange interpolation

**TOTPValidator:**
- Purpose: Generates and validates TOTP codes per RFC 6238
- Examples: `twofa/internal/services/twofaService/totp/totp.go`
- Pattern: Time-based one-time password with 30-second windows

**PGStorage (Repository):**
- Purpose: Abstract database access using pgx directly (no ORM)
- Examples: `<service>/internal/storage/pgstorage/pgstorage.go`, `user.go`, `session.go`
- Pattern: Separate file per entity, constructor with connection pool, methods for CRUD

**RedisStorage:**
- Purpose: Manage session tokens and rate limit counters with TTL
- Examples: `<service>/internal/storage/redisstorage/redisstorage.go`, `session.go`
- Pattern: Wrapper around redis/go-redis client with TTL management

**MPCNodeService:**
- Purpose: Store, retrieve, and delete encrypted shares with access control
- Examples: `mpc/internal/services/mpcService/mpc_service.go`, `store_share.go`, `retrieve_share.go`
- Pattern: Delegates encryption/decryption to storage layer, enforces shared-secret authorization

## Entry Points

**Gateway Service:**
- Location: `gateway/cmd/app/main.go`
- Triggers: Process startup, listens on HTTP port (default 8080)
- Responsibilities:
  - Bind HTTP REST routes to gRPC handlers
  - Rate limiting middleware
  - CORS and request logging
  - Token validation via Auth Service
  - Route to Auth or TwoFA services

**Auth Service:**
- Location: `auth/cmd/app/main.go`
- Triggers: Process startup, listens on gRPC port (default 9090)
- Responsibilities:
  - Initialize PostgreSQL pool, Redis client, Kafka producer
  - Register gRPC service and interceptors
  - Start graceful shutdown handler
  - Expose health check endpoint

**TwoFA Service:**
- Location: `twofa/cmd/app/main.go`
- Triggers: Process startup, listens on gRPC port (default 9091)
- Responsibilities:
  - Initialize PostgreSQL pool, Redis client, Kafka producer, MPC node clients
  - Register gRPC service and interceptors
  - Start graceful shutdown handler
  - Expose health check endpoint

**MPC Node:**
- Location: `mpc/cmd/app/main.go`
- Triggers: Process startup, listens on gRPC port (configurable per node)
- Responsibilities:
  - Load encryption key from config
  - Initialize PostgreSQL pool for share storage
  - Register gRPC service with shared-secret authorization interceptor
  - Start graceful shutdown handler
  - Expose health check endpoint

## Error Handling

**Strategy:** gRPC status codes for structured error propagation

**Patterns:**
- `InvalidArgument`: Validation failures (bad email format, weak password, invalid OTP)
- `NotFound`: Resource not found (user, 2FA record, share)
- `AlreadyExists`: Duplicate creation attempts (email already registered, 2FA already enabled)
- `Unauthenticated`: Token validation failures, authorization failures
- `PermissionDenied`: User doesn't have permission to access resource
- `FailedPrecondition`: Operation preconditions not met (2FA not enabled for disable, share count < 2 for combine)
- `Internal`: Unexpected errors (database errors, encryption errors)

All errors logged with slog at appropriate levels (debug, info, warn, error) without exposing sensitive data (passwords, keys, secrets).

## Cross-Cutting Concerns

**Logging:** slog structured logging with fields (user_id, operation, duration, status)
- **Never log:** passwords, TOTP secrets, encryption keys, share data
- **Always log:** user_id, operation name, error type, timing information
- Implementation: gRPC unary/stream interceptors in `<service>/internal/middleware/interceptors.go`

**Validation:**
- Auth Service: password policy (length, character classes, no sequences)
- TwoFA Service: OTP format, rate limiting
- All handlers: required field checks, email format validation
- Implementation: Validator functions in service layer before persistence

**Authentication:**
- JWT (RS256) with 15-minute access token expiry
- Refresh token rotation with 7-day TTL in Redis
- gRPC metadata inspection for Authorization header
- Implementation: Auth Service ValidateToken method, interceptor for token validation

**Rate Limiting:**
- Gateway: per-IP general rate limiting
- TwoFA: per-user 2FA verification attempts (5 per 5 minutes)
- Implementation: Redis counters with Lua scripts or simple increment/expire

**Audit Trail:**
- All operations published to Kafka: `<domain>.<operation>` topics
- Payload: user_id, operation, timestamp, status (never secret data)
- Consumer: Separate audit service (future) or log sink

**Metrics (Prometheus):**
- `auth_requests_total{method, status}`: Auth operation counters
- `auth_request_duration_seconds`: Auth latency histogram
- `twofa_operations_total{operation, status}`: TwoFA operation counters
- `twofa_mpc_latency_seconds{node_id}`: MPC node request latency
- `mpc_operations_total{node_id, operation, status}`: MPC node operation counters
- Implementation: Interceptor middleware wrapping handler calls

---

*Architecture analysis: 2026-04-11*
