# Roadmap: MPC-2FA

## Overview

Build a two-factor authentication system with distributed secret storage across three Go microservices (Auth, TwoFA, MPC). Start with project scaffolding and the Auth service foundation, then implement cryptographic primitives (Shamir and TOTP) as standalone tested modules, build the MPC node service for encrypted share storage, wire everything together through the TwoFA orchestration service, and finish with cross-service hardening (health checks, monitoring, audit).

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 1: Project Scaffolding** - Go modules, proto generation, Docker Compose, config pattern, Clean Architecture skeleton for all 3 services
- [ ] **Phase 2: Auth Registration** - User registration with password validation (bcrypt, sequential char detection) and unit tests
- [ ] **Phase 3: Auth Sessions & JWT** - Login, JWT RS256 tokens, refresh rotation with theft detection, logout, token validation
- [ ] **Phase 4: Shamir Secret Sharing** - GF(256) arithmetic, split/combine in pure Go, comprehensive tests
- [ ] **Phase 5: TOTP Implementation** - RFC 6238 TOTP generation and validation, provisioning URI, time window tests
- [ ] **Phase 6: MPC Node Service** - Encrypted share storage with AES-256-GCM, gRPC auth interceptor, CRUD operations
- [ ] **Phase 7: TwoFA Setup Flow** - Orchestrate 2FA setup: generate secret, Shamir split, distribute to MPC nodes, backup codes
- [ ] **Phase 8: TwoFA Verification & Management** - OTP verification, rate limiting, disable 2FA, status check, single-use enforcement
- [ ] **Phase 9: Cross-Service Hardening** - Health checks, graceful shutdown, Prometheus metrics, structured logging, Kafka audit, error sanitization

## Phase Details

### Phase 1: Project Scaffolding
**Goal**: All three services have runnable skeletons with Clean Architecture structure, proto generation, and local infrastructure
**Depends on**: Nothing (first phase)
**Requirements**: INFRA-01, INFRA-02, INFRA-08, INFRA-09, INFRA-10, INFRA-11
**Success Criteria** (what must be TRUE):
  1. Each service (auth, twofa, mpc) is a separate Go module that compiles successfully
  2. Running `generate.sh` produces Go code from proto definitions in each service
  3. `docker-compose up` starts PostgreSQL and Redis for local development
  4. Each service starts, loads config.yaml, and listens on its gRPC port
  5. Bootstrap factories wire dependencies through interfaces (handler -> service -> repository)
**Plans**: 6 plans

Plans:
- [x] 01-01-PLAN.md — Auth Go module, proto definitions, generate.sh, Makefile
- [x] 01-02-PLAN.md — TwoFA Go module, proto definitions, generate.sh, Makefile
- [x] 01-03-PLAN.md — MPC Go module, proto definitions, generate.sh, Makefile
- [x] 01-04-PLAN.md — Auth Docker Compose, config, Clean Architecture skeleton
- [x] 01-05-PLAN.md — TwoFA Docker Compose, config, Clean Architecture skeleton
- [x] 01-06-PLAN.md — MPC Docker Compose, config, Clean Architecture skeleton

### Phase 2: Auth Registration
**Goal**: Users can create accounts with strongly validated passwords
**Depends on**: Phase 1
**Requirements**: AUTH-01, AUTH-02, AUTH-08
**Success Criteria** (what must be TRUE):
  1. User can register with email and password via gRPC and the account is persisted in PostgreSQL
  2. Password below 12 chars or missing any required character class is rejected with clear error
  3. Password containing 4+ sequential characters (1234, abcd, qwer, dcba) is rejected
  4. Unit tests cover every password validation rule including boundary cases (3 vs 4 sequential)
**Plans**: 2 plans

Plans:
- [x] 02-01-PLAN.md — Password validation with TDD (sequential/repeated char detection, error types, 20+ boundary tests)
- [x] 02-02-PLAN.md — Registration flow (Storage interface refactor, CreateUser/GetUserByEmail, Register service+handler, mocked tests)

### Phase 3: Auth Sessions & JWT
**Goal**: Users can authenticate, maintain sessions, and other services can validate their identity
**Depends on**: Phase 2
**Requirements**: AUTH-03, AUTH-04, AUTH-05, AUTH-06, AUTH-07, SEC-01, SEC-03
**Success Criteria** (what must be TRUE):
  1. User can login with email/password and receive an RS256 JWT access token (15min) and refresh token (7 days in Redis)
  2. User can refresh their access token; old refresh token is deleted and new one issued (rotation)
  3. Reusing a previously rotated refresh token revokes ALL tokens for that user (theft detection)
  4. User can logout and their refresh token is deleted from Redis
  5. Another service can validate an access token and receive user_id and claims; algorithm confusion (non-RS256) is rejected
**Plans**: 3 plans

Plans:
- [ ] 03-01-PLAN.md — JWT infrastructure: RS256 token helper, SessionStorage interface, Redis three-key implementation, domain errors, mock generation
- [ ] 03-02-PLAN.md — Session service methods: Login, RefreshToken with theft detection, ValidateToken, Logout, LogoutAll, Register auto-login, all with unit tests
- [ ] 03-03-PLAN.md — gRPC handlers: Login, RefreshToken, Logout, LogoutAll, ValidateToken, Register update with token population

### Phase 4: Shamir Secret Sharing
**Goal**: A tested, from-scratch Shamir Secret Sharing library operates correctly in GF(256)
**Depends on**: Phase 1
**Requirements**: CRYPTO-01, CRYPTO-02, CRYPTO-03
**Success Criteria** (what must be TRUE):
  1. Split(secret, n=3, threshold=2) produces 3 distinct shares from any input byte sequence
  2. Combine with any 2-of-3 shares recovers the original secret exactly
  3. Combine with only 1-of-3 shares does NOT recover the secret
  4. GF(256) arithmetic passes property tests (associativity, commutativity, distributivity)
**Plans**: 2 plans

Plans:
- [ ] 04-01: TBD

### Phase 5: TOTP Implementation
**Goal**: A tested TOTP library generates and validates one-time passwords per RFC 6238
**Depends on**: Phase 1
**Requirements**: CRYPTO-04, CRYPTO-05, CRYPTO-06, CRYPTO-07
**Success Criteria** (what must be TRUE):
  1. GenerateSecret produces a 20-byte random secret encoded in base32
  2. GenerateOTP produces a 6-digit code using SHA-1 with 30-second periods matching RFC 6238 test vectors
  3. ValidateOTP accepts codes from the current time step and +-1 adjacent windows
  4. GenerateProvisioningURI returns a valid otpauth://totp/... URI with issuer, account, and secret
**Plans**: 2 plans

Plans:
- [ ] 05-01: TBD

### Phase 6: MPC Node Service
**Goal**: Each MPC node can securely store, retrieve, and delete encrypted secret shares with access control
**Depends on**: Phase 1
**Requirements**: MPC-01, MPC-02, MPC-03, MPC-04, MPC-05, MPC-06
**Success Criteria** (what must be TRUE):
  1. StoreShare encrypts share data with AES-256-GCM (unique nonce via crypto/rand) and persists encrypted_data + nonce in PostgreSQL
  2. RetrieveShare decrypts and returns the original share data
  3. DeleteShare removes all shares for a given user from the node
  4. Storing a duplicate (user_id, share_index) is rejected by unique constraint
  5. Requests without valid shared secret in gRPC metadata are rejected by the interceptor
**Plans**: 2 plans

Plans:
- [ ] 06-01: TBD
- [ ] 06-02: TBD

### Phase 7: TwoFA Setup Flow
**Goal**: Users can enable 2FA with their TOTP secret securely split and distributed across MPC nodes
**Depends on**: Phase 3, Phase 4, Phase 5, Phase 6
**Requirements**: 2FA-01, 2FA-02, 2FA-08, SEC-04
**Success Criteria** (what must be TRUE):
  1. User calls Setup2FA and receives a provisioning URI and 10 backup codes
  2. The TOTP secret is split into 3 shares and all 3 are stored across MPC nodes; if any node is unreachable, setup fails completely
  3. The TOTP secret is zeroized from memory after share distribution — never persisted whole
  4. Backup codes are bcrypt-hashed before storage in PostgreSQL
**Plans**: 2 plans

Plans:
- [ ] 07-01: TBD
- [ ] 07-02: TBD

### Phase 8: TwoFA Verification & Management
**Goal**: Users can verify OTP codes, manage their 2FA status, and the system enforces security constraints
**Depends on**: Phase 7
**Requirements**: 2FA-03, 2FA-04, 2FA-05, 2FA-06, 2FA-07, 2FA-09, SEC-05
**Success Criteria** (what must be TRUE):
  1. User can verify OTP: 2 shares retrieved from MPC nodes, Shamir-combined, TOTP validated (+-1 window), secret zeroized
  2. First successful verification enables 2FA (is_enabled transitions to true)
  3. More than 5 failed verification attempts within 5 minutes are rejected (rate limiting via Redis)
  4. User can disable 2FA (requires valid OTP first, then shares deleted from all nodes and metadata removed)
  5. User can check 2FA status (is_enabled, created_at) and OTP reuse within the same time window is rejected
**Plans**: 2 plans

Plans:
- [ ] 08-01: TBD
- [ ] 08-02: TBD

### Phase 9: Cross-Service Hardening
**Goal**: All services meet production-readiness standards for observability, reliability, and security hygiene
**Depends on**: Phase 8
**Requirements**: INFRA-03, INFRA-04, INFRA-05, INFRA-06, INFRA-07, SEC-02
**Success Criteria** (what must be TRUE):
  1. Each service responds to gRPC health check requests (serving/not-serving)
  2. Stopping a service triggers ordered teardown: gRPC stop, Kafka flush, Redis close, PostgreSQL close — no resource leaks
  3. Each service exposes Prometheus metrics (request count, duration, service-specific counters) on a metrics endpoint
  4. Log output is structured JSON via slog; grep for passwords/secrets/shares/keys returns zero matches
  5. Each service publishes audit events to Kafka (user_id, operation, timestamp) with no secret data; gRPC errors contain no internal state
**Plans**: 2 plans

Plans:
- [ ] 09-01: TBD
- [ ] 09-02: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7 -> 8 -> 9
Note: Phases 4, 5, 6 can execute in parallel after Phase 1 (no mutual dependencies). Phase 2 -> 3 is sequential. Phase 7 requires 3, 4, 5, 6 all complete.

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Project Scaffolding | 0/2 | Not started | - |
| 2. Auth Registration | 0/2 | Not started | - |
| 3. Auth Sessions & JWT | 0/3 | Not started | - |
| 4. Shamir Secret Sharing | 0/1 | Not started | - |
| 5. TOTP Implementation | 0/1 | Not started | - |
| 6. MPC Node Service | 0/2 | Not started | - |
| 7. TwoFA Setup Flow | 0/2 | Not started | - |
| 8. TwoFA Verification & Management | 0/2 | Not started | - |
| 9. Cross-Service Hardening | 0/2 | Not started | - |
