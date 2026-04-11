# External Integrations

**Analysis Date:** 2026-04-11

## APIs & External Services

**None - this is a closed system.** No external APIs (Stripe, OAuth, email services) are integrated. All authentication is custom-built.

## Data Storage

**Databases:**
- PostgreSQL (latest)
  - Connection: Configured in `config/config.go` for each service
  - Driver: `github.com/jackc/pgx/v5` (direct SQL, no ORM)
  - Shared schema across all services:
    - `users` - User accounts, email, password hashes (Auth Service)
    - `sessions` - Session audit logs (Auth Service)
    - `2fa_metadata` - 2FA setup status, TOTP backup codes metadata (TwoFA Service)
    - `shares` - Encrypted secret shares, MPC node assignments (all MPC Nodes)
    - `audit_log` - Central audit trail (all services)
  - Initialization: Each service runs `initTables()` on startup to ensure schema

**File Storage:**
- None - local filesystem only

**Caching:**
- Redis 8.6.2
  - Connection: Configured in `config/config.go`
  - Client: `github.com/redis/go-redis/v9`
  - Usage by service:
    - **Auth Service**: Stores refresh tokens with TTL 7 days (key pattern: `refresh_token:{user_id}`)
    - **Gateway**: Stores rate limit counters (key pattern: `rate_limit:{ip_or_user_id}`, TTL 5 minutes)
  - No session caching - sessions are audit-logged in PostgreSQL
  - No TOTP secret caching - TOTP secrets never persisted, only transient in memory

## Authentication & Identity

**Auth Provider:**
- Custom JWT implementation
  - Framework: `github.com/golang-jwt/jwt/v5`
  - Algorithm: RS256 (RSA with SHA-256)
  - Token types:
    - **Access token**: 15 minutes validity, contains user_id and standard claims
    - **Refresh token**: 7 days validity, stored in Redis with TTL
  - Key management: Private key for signing, public key for validation (loaded in `internal/services/authService/jwt.go`)
  - Validation: `Auth.ValidateToken` RPC called by Gateway and other services

**Password Hashing:**
- bcrypt with cost=12 (`golang.org/x/crypto/bcrypt`)
- Password validation rules enforced before hashing in `internal/services/authService/password_validation.go`:
  - Minimum 12 characters
  - At least 1 lowercase (a-z), 1 uppercase (A-Z), 1 digit (0-9), 1 special character
  - No 4+ consecutive characters in sequence (numeric, alphabetic, or keyboard rows)

**No OAuth/SSO:** Not implemented per requirements.

## Monitoring & Observability

**Error Tracking:**
- None (no external service)
- Structured logging with slog (Go stdlib)

**Logs:**
- Approach: Structured logging via `log/slog` package
- Interceptors in each service capture: request method, duration, status code, user_id (when available)
- Security: Passwords, secret shares, TOTP secrets, encryption keys NEVER logged
- Output: Stdout (captured by container orchestration)
- Levels: DEBUG (development), INFO (default), WARN (errors), ERROR (failures)

**Metrics:**
- Prometheus (no external SaaS)
  - Framework: `github.com/prometheus/client_golang`
  - Scrape target: Each service exposes `/metrics` on designated Prometheus port
  - Metrics collected per service:
    - **Common**: request latency histogram, request counter (method, status code)
    - **Auth**: registration attempts, login attempts, token refresh count
    - **TwoFA**: setup/verify/disable operations, MPC node communication latency
    - **MPC**: share store/retrieve operations, encryption/decryption time
  - Retention: Configured in monitoring/prometheus.yml
  - Visualization: Grafana (local, not SaaS)

## CI/CD & Deployment

**Hosting:**
- Docker containers (one per service instance)
- Kubernetes-ready (gRPC Health Check Protocol in each service)
- No cloud provider specified (on-premises or managed Kubernetes)

**CI Pipeline:**
- None configured yet (can use GitHub Actions, GitLab CI, or Jenkins)

**Local Development:**
- `docker-compose.yaml` in each service directory:
  - Spins up PostgreSQL, Redis, Kafka for that service
  - Optionally Prometheus and Grafana

## Environment Configuration

**Required env vars:**
Each service's `config/config.go` loads these from YAML or environment:

**Auth Service:**
- `DATABASE_URL` or config.yaml `postgres.dsn` - PostgreSQL connection string
- `REDIS_URL` or config.yaml `redis.addr` - Redis address
- `KAFKA_BROKERS` - Comma-separated list (e.g., "localhost:9092")
- `JWT_PRIVATE_KEY_PATH` - Path to RSA private key for signing
- `JWT_PUBLIC_KEY_PATH` - Path to RSA public key for validation
- `PORT` - gRPC server port (default 50051)
- `ENVIRONMENT` - "development" or "production"

**Gateway:**
- `DATABASE_URL` - For rate limit validation
- `REDIS_URL` - For rate limit counters
- `AUTH_SERVICE_ADDR` - "auth-service:50051" (gRPC endpoint)
- `TWOFA_SERVICE_ADDR` - "twofa-service:50052" (gRPC endpoint)
- `PORT` - HTTP server port (default 8080)
- `JWT_PUBLIC_KEY_PATH` - Shared with Auth Service

**TwoFA Service:**
- `DATABASE_URL` - PostgreSQL
- `KAFKA_BROKERS` - Kafka
- `MPC_NODE_ADDRS` - "mpc-node-1:50053,mpc-node-2:50054,mpc-node-3:50055" (3 gRPC addresses)
- `PORT` - gRPC server port (default 50052)

**MPC Nodes (×3):**
- `DATABASE_URL` - PostgreSQL
- `KAFKA_BROKERS` - Kafka
- `ENCRYPTION_KEY` - AES-256-GCM key for at-rest encryption of shares (loaded as env var or config)
- `NODE_ID` - "1", "2", or "3" (identifies which share this node holds)
- `PORT` - gRPC server port (50053, 50054, 50055 respectively)

**Secrets location:**
- Development: Environment variables or `config.yaml` (NOT committed)
- Production: Kubernetes Secrets or external secrets management (HashiCorp Vault, AWS Secrets Manager)
- Private keys (JWT): Loaded from filesystem paths specified in config

## Webhooks & Callbacks

**Incoming:**
- None (this is not a platform for third-party integrations)

**Outgoing:**
- **Kafka events** (internal event stream):
  - Produced by: All services (Auth, TwoFA, MPC)
  - Topics:
    - `user.registered` - User registration (Auth Service)
    - `user.logged_in` - User login (Auth Service)
    - `token.refreshed` - Token refresh (Auth Service)
    - `2fa.setup_initiated` - 2FA setup started (TwoFA Service)
    - `2fa.setup_completed` - 2FA enabled (TwoFA Service)
    - `2fa.verified` - User verified 2FA code (TwoFA Service)
    - `2fa.disabled` - 2FA disabled (TwoFA Service)
    - `share.stored` - Share stored in MPC node (MPC Nodes)
    - `share.retrieved` - Share retrieved (MPC Nodes)
  - Event payload: `{user_id, operation, timestamp, ...}` - Never includes shares or secrets
  - Consumer: None configured yet (audit log aggregation in future)

## Encryption & Secrets Management

**At-Rest Encryption:**
- MPC Nodes: AES-256-GCM for storing secret shares
  - Implementation: `golang.org/x/crypto/aes` and `crypto/cipher`
  - Nonce: Generated per-operation via `crypto.rand`
  - Key: `ENCRYPTION_KEY` environment variable or config

**Transport Encryption:**
- gRPC over TLS (configured in each service)
- HTTPS from Frontend to API Gateway

**Transient Data:**
- TOTP secrets: Never persisted, only held in memory during operations, zeroed after use (`internal/services/twofa/`)
- Shamir shares in memory: Zeroized after combine operation
- Passwords: Never stored in logs or responses

---

*Integration audit: 2026-04-11*
