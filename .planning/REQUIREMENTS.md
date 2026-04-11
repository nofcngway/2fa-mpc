# Requirements: MPC-2FA

**Defined:** 2026-04-11
**Core Value:** TOTP-секрет никогда не существует в персистентном хранилище целиком — безопасность через распределение и шифрование долей.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Authentication

- [ ] **AUTH-01**: User can register with email and password (bcrypt cost=12)
- [ ] **AUTH-02**: Password validated before hashing — min 12 chars, 1 lowercase, 1 uppercase, 1 digit, 1 special char, no 4+ sequential chars (ASCII/keyboard)
- [ ] **AUTH-03**: User can login and receive JWT access token (RS256, 15min) and refresh token (7 days, stored in Redis)
- [ ] **AUTH-04**: User can refresh access token via refresh token with rotation (old token deleted, new issued)
- [ ] **AUTH-05**: Refresh token reuse detected — revoke all tokens for user (theft detection)
- [ ] **AUTH-06**: User can logout (refresh token deleted from Redis, session invalidated)
- [ ] **AUTH-07**: Access token can be validated by other services (returns user_id and claims)
- [ ] **AUTH-08**: Password validation has unit tests covering each rule and boundary cases (3 vs 4 sequential chars)

### TwoFA Orchestration

- [ ] **2FA-01**: User can setup 2FA — TOTP secret generated, split via Shamir (2-of-3), shares sent to 3 MPC nodes, secret zeroized, provisioning URI returned
- [ ] **2FA-02**: Setup fails if any MPC node is unreachable (all 3 shares MUST be stored)
- [ ] **2FA-03**: User can verify OTP — 2 shares retrieved from MPC nodes, Shamir combine, TOTP validation (+-1 window), secret zeroized
- [ ] **2FA-04**: First successful verification enables 2FA (is_enabled=true)
- [ ] **2FA-05**: OTP verification rate limited — max 5 attempts per 5 minutes per user_id (Redis)
- [ ] **2FA-06**: User can disable 2FA — verify OTP first, then delete shares from all 3 nodes and metadata from PostgreSQL
- [ ] **2FA-07**: User can check 2FA status (is_enabled, created_at)
- [ ] **2FA-08**: 10 backup codes generated on setup, each bcrypt-hashed, stored in PostgreSQL
- [ ] **2FA-09**: OTP single-use enforcement — store last-used time counter per user, reject reuse within same window

### Cryptographic Core

- [ ] **CRYPTO-01**: Shamir Secret Sharing implemented from scratch — Split(secret, n=3, threshold=2) and Combine(shares) in GF(256)
- [ ] **CRYPTO-02**: GF(256) arithmetic — addition via XOR, multiplication via log/exp tables, polynomial evaluation
- [ ] **CRYPTO-03**: Shamir unit tests — split→combine roundtrip, any 2-of-3 recovers, 1-of-3 does NOT recover
- [ ] **CRYPTO-04**: TOTP implementation per RFC 6238 — SHA-1, 6 digits, 30s period, base32 secret (20 bytes)
- [ ] **CRYPTO-05**: TOTP generates valid provisioning URI (otpauth://totp/...)
- [ ] **CRYPTO-06**: TOTP validation allows +-1 time window
- [ ] **CRYPTO-07**: TOTP unit tests — generation, validation, time window edge cases

### MPC Node

- [ ] **MPC-01**: StoreShare — encrypt share data with AES-256-GCM (unique nonce via crypto/rand), store encrypted_data + nonce in PostgreSQL
- [ ] **MPC-02**: RetrieveShare — read encrypted_data + nonce, decrypt, return share data
- [ ] **MPC-03**: DeleteShare — delete all shares for a user from this node
- [ ] **MPC-04**: Unique constraint on (user_id, share_index) per node
- [ ] **MPC-05**: gRPC interceptor validates shared secret via metadata ("authorization" header)
- [ ] **MPC-06**: AES-256-GCM encryption key loaded from config (ENCRYPTION_KEY), nonce never reused

### Infrastructure

- [ ] **INFRA-01**: Each service follows Clean Architecture — handler → service → repository, dependencies via interfaces
- [ ] **INFRA-02**: DI through bootstrap factories in internal/bootstrap/
- [ ] **INFRA-03**: gRPC Health Check Protocol in each service
- [ ] **INFRA-04**: Graceful shutdown with ordered teardown (gRPC stop → Kafka flush → Redis close → PG close)
- [ ] **INFRA-05**: Prometheus metrics per service (requests total, duration, service-specific counters)
- [ ] **INFRA-06**: Structured logging with slog — secrets, passwords, shares, encryption keys NEVER logged
- [ ] **INFRA-07**: Kafka audit events per service (user_id, operation, timestamp — no secret data)
- [ ] **INFRA-08**: Configuration via config.yaml loaded in config/config.go
- [ ] **INFRA-09**: Proto definitions in api/ with generate.sh for protobuf code generation
- [ ] **INFRA-10**: Each service is separate Go module (github.com/vbncursed/vkr/{auth,twofa,mpc})
- [ ] **INFRA-11**: Docker Compose per service for local dependencies (PostgreSQL, Redis)

### Security

- [ ] **SEC-01**: JWT validation uses WithValidMethods([]string{"RS256"}) — prevents algorithm confusion attack
- [ ] **SEC-02**: gRPC errors sanitized — no internal state leaked in error messages
- [ ] **SEC-03**: Passwords never returned in responses or logged
- [ ] **SEC-04**: TOTP secret never persisted — only transient in memory, zeroized after use
- [ ] **SEC-05**: Share data and encryption keys never logged or included in Kafka events

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Gateway

- **GW-01**: API Gateway with REST→gRPC translation
- **GW-02**: Rate limiting at gateway level
- **GW-03**: JWT validation middleware

### Frontend

- **FE-01**: Next.js frontend for registration/login
- **FE-02**: 2FA setup flow with QR code display
- **FE-03**: OTP verification UI

### Monitoring

- **MON-01**: Prometheus + Grafana dashboard configuration
- **MON-02**: Alerting rules for critical metrics

### Advanced Security

- **ASEC-01**: mTLS between services (replace shared secret)
- **ASEC-02**: Secret zeroization via runtime/secret (Linux-only, experimental)

## Out of Scope

| Feature | Reason |
|---------|--------|
| OAuth / SSO | Not in  scope |
| Email verification | Not in  scope |
| ORM (GORM etc.) | Architecture decision — pgx only |
| HTTP endpoints in services | gRPC only (HTTP in future Gateway) |
| Third-party Shamir libraries | Academic requirement — implement from scratch |
| Mobile app | Web-first, academic project |
| Real-time notifications | Not required for 2FA system |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| AUTH-01 | Phase 2 | Pending |
| AUTH-02 | Phase 2 | Pending |
| AUTH-03 | Phase 3 | Pending |
| AUTH-04 | Phase 3 | Pending |
| AUTH-05 | Phase 3 | Pending |
| AUTH-06 | Phase 3 | Pending |
| AUTH-07 | Phase 3 | Pending |
| AUTH-08 | Phase 2 | Pending |
| 2FA-01 | Phase 7 | Pending |
| 2FA-02 | Phase 7 | Pending |
| 2FA-03 | Phase 8 | Pending |
| 2FA-04 | Phase 8 | Pending |
| 2FA-05 | Phase 8 | Pending |
| 2FA-06 | Phase 8 | Pending |
| 2FA-07 | Phase 8 | Pending |
| 2FA-08 | Phase 7 | Pending |
| 2FA-09 | Phase 8 | Pending |
| CRYPTO-01 | Phase 4 | Pending |
| CRYPTO-02 | Phase 4 | Pending |
| CRYPTO-03 | Phase 4 | Pending |
| CRYPTO-04 | Phase 5 | Pending |
| CRYPTO-05 | Phase 5 | Pending |
| CRYPTO-06 | Phase 5 | Pending |
| CRYPTO-07 | Phase 5 | Pending |
| MPC-01 | Phase 6 | Pending |
| MPC-02 | Phase 6 | Pending |
| MPC-03 | Phase 6 | Pending |
| MPC-04 | Phase 6 | Pending |
| MPC-05 | Phase 6 | Pending |
| MPC-06 | Phase 6 | Pending |
| INFRA-01 | Phase 1 | Pending |
| INFRA-02 | Phase 1 | Pending |
| INFRA-03 | Phase 9 | Pending |
| INFRA-04 | Phase 9 | Pending |
| INFRA-05 | Phase 9 | Pending |
| INFRA-06 | Phase 9 | Pending |
| INFRA-07 | Phase 9 | Pending |
| INFRA-08 | Phase 1 | Pending |
| INFRA-09 | Phase 1 | Pending |
| INFRA-10 | Phase 1 | Pending |
| INFRA-11 | Phase 1 | Pending |
| SEC-01 | Phase 3 | Pending |
| SEC-02 | Phase 9 | Pending |
| SEC-03 | Phase 3 | Pending |
| SEC-04 | Phase 7 | Pending |
| SEC-05 | Phase 8 | Pending |

**Coverage:**
- v1 requirements: 46 total
- Mapped to phases: 46
- Unmapped: 0

---
*Requirements defined: 2026-04-11*
*Last updated: 2026-04-11 after roadmap creation*
