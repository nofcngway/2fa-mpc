# Technology Stack

**Analysis Date:** 2026-04-11

## Languages

**Primary:**
- Go 1.26.2 - All backend services (gateway, auth, twofa, mpc)

**Frontend:**
- Next.js - Web UI (separate repository, communicates via HTTPS REST)

## Runtime

**Environment:**
- Go runtime 1.26.2

**Package Manager:**
- Go modules (go.mod / go.sum per service)

## Frameworks

**RPC Framework:**
- gRPC (google.golang.org/grpc) - Inter-service communication between Gateway, Auth Service, TwoFA Service, and MPC Nodes

**Protocol Definitions:**
- Protocol Buffers (protobuf) - Service contracts and message definitions
- Google API Annotations - HTTP→gRPC transcoding in Gateway

**Web Server (Gateway only):**
- HTTP REST API - Single entry point from Next.js frontend
- Net/http standard library with gRPC-Gateway for REST→gRPC translation

## Key Dependencies

**Critical:**
- `google.golang.org/grpc` - Core RPC framework for inter-service communication
- `github.com/jackc/pgx/v5` - PostgreSQL driver (raw SQL, no ORM)
- `github.com/redis/go-redis/v9` - Redis client for session storage and rate limiting
- `github.com/segmentio/kafka-go` - Kafka client for audit events
- `github.com/golang-jwt/jwt/v5` - JWT token generation and validation (RS256)
- `golang.org/x/crypto` - Cryptographic operations: bcrypt (password hashing), AES-256-GCM (share encryption)

**Monitoring & Observability:**
- `github.com/prometheus/client_golang` - Prometheus metrics collection

**Utilities:**
- `github.com/google/uuid` - UUID generation for user and session IDs
- `gopkg.in/yaml.v3` - Configuration file parsing (config.yaml)

## Configuration

**Environment:**
- `config.yaml` per service - Configuration loaded via `internal/config/config.go`
- Required config sections per service:
  - Database: PostgreSQL DSN, connection pool settings
  - Redis: Connection string, TTL settings (refresh tokens 7 days, rate limit window 5 minutes)
  - Kafka: Broker addresses, topic names for audit events
  - Server: Port, gRPC settings, timeouts
  - JWT: RS256 private/public key paths, TTL settings (access 15m, refresh 7d)
  - MPC (TwoFA Service only): 3 MPC node addresses, gRPC client timeouts (5s)
  - Encryption (MPC Node only): AES-256 encryption key for share storage

**Environment Variables:**
- Not explicitly documented, but config.yaml loading pattern suggests ENCRYPTION_KEY, NODE_ID (for MPC nodes)

**Build:**
- Makefile per service - Build, proto generation, Docker targets
- `scripts/generate.sh` per service - Protocol Buffer code generation
- `docker-compose.yaml` per service - Local development: PostgreSQL, Redis

## Service Structure

**API Gateway** (`gateway/`):
- HTTP REST endpoint for Next.js frontend
- gRPC clients to Auth and TwoFA services
- Rate limiting via Redis
- Request routing and authentication validation

**Auth Service** (`auth/`):
- User registration and login
- JWT token generation (RS256: access 15m, refresh 7d)
- Password validation and bcrypt hashing (cost=12)
- Session management via Redis
- PostgreSQL user/session storage

**TwoFA Service** (`twofa/`):
- TOTP provisioning and verification
- Shamir Secret Sharing orchestration (2-of-3 split)
- gRPC clients to 3 MPC nodes for share distribution
- Backup code generation and storage
- Rate limiting for 2FA attempts (5 per 5 minutes per user)

**MPC Node** (`mpc/` × 3 nodes):
- Share storage with AES-256-GCM encryption at rest
- gRPC endpoints for StoreShare, RetrieveShare, DeleteShare
- Per-node PostgreSQL database

## Data Storage

**PostgreSQL:**
- Databases per service (or shared, depending on deployment)
- Tables:
  - `users` (auth): id, email, password_hash, created_at
  - `sessions` (auth): id, user_id, created_at, expires_at
  - `audit_log` (all services): user_id, operation, timestamp, service
  - `twofa_records` (twofa): user_id, is_enabled, created_at
  - `backup_codes` (twofa): user_id, code_hash
  - `shares` (mpc): id, user_id, share_index, encrypted_data, nonce, created_at
- No ORM (pgx used directly)
- Initialization via `initTables()` in each service's storage layer

**Redis:**
- Refresh token storage: `refresh_token:{token_jti}` → user_id with TTL 7 days
- Rate limit counters: `rate_limit:{user_id}:{operation}` with TTL 5 minutes

**Kafka:**
- Audit topics (one per service or unified):
  - `user.registered`, `user.logged_in` (auth)
  - `2fa.verified`, `2fa.disabled`, `token.refreshed` (twofa)
  - `share.stored`, `share.retrieved`, `share.deleted` (mpc)
- Events contain: user_id, operation, timestamp, node_id (mpc only)
- Never contains: passwords, TOTP secrets, share data, encryption keys

## Cryptography

**Password Hashing:**
- bcrypt, cost=12

**Token Signing:**
- RS256 (RSA-2048 + SHA-256)
- Access token: 15 minutes
- Refresh token: 7 days (stored in Redis)
- Keys managed via config.yaml (private/public key paths)

**Secret Distribution:**
- Shamir Secret Sharing (custom implementation, GF(256), 2-of-3)
- Implementation location: `twofa/internal/services/twofaService/shamir/`

**Share Encryption:**
- AES-256-GCM at rest in MPC nodes
- Nonce: 12 bytes, generated via crypto/rand per operation
- Key: 32 bytes from config.yaml (ENCRYPTION_KEY)

**TOTP:**
- RFC 6238 compliant
- 6-digit codes, 30-second window
- Validation: ±1 time window tolerance
- Secret: 20 bytes, base32 encoded
- Never persisted; only exists in memory during verify/setup

## Deployment

**Containerization:**
- Docker Compose files per service (PostgreSQL, Redis, Kafka)
- Service containerization via Dockerfile (not shown in current codebase state)

**Observability:**
- Prometheus metrics: request counts, durations, errors per service
- Grafana dashboards (configuration in `monitoring/` directory)
- Structured logging: slog (standard library)
- Never log: passwords, TOTP secrets, share data, encryption keys

**Graceful Shutdown:**
- All services: proper connection closure (PostgreSQL, Redis, Kafka, gRPC listeners)
- Health check: gRPC Health Check Protocol enabled on all services

---

*Stack analysis: 2026-04-11*
