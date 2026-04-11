# External Integrations

**Analysis Date:** 2026-04-11

## APIs & External Services

**None (Internal System Only):**
- No third-party authentication (OAuth, SSO)
- No email verification service
- No SMS/push notification service
- System integrates only internal services via gRPC

## Data Storage

**Databases:**

**PostgreSQL:**
- Multi-instance deployment (one per service or shared)
- Connection: Standard PostgreSQL connection string in `config.yaml`
- Client: pgx/v5 (direct SQL, no ORM)
- Tables per service:
  - `auth/`: users (email, password_hash), sessions (refresh state)
  - `twofa/`: twofa_records (2FA metadata), backup_codes (hashed)
  - `mpc/`: shares (encrypted TOTP share fragments)
- All services: audit_log table for compliance
- Initialization: Each service calls `initTables()` on startup via `internal/storage/pgstorage/pgstorage.go`

**Redis:**
- Single or replicated instance
- Connection: Standard Redis URL in `config.yaml`
- Client: github.com/redis/go-redis/v9
- Data structures:
  - Refresh tokens: String keys with 7-day TTL
  - Rate limit counters: String/atomic increment with 5-minute TTL
- Usage:
  - Auth Service: `internal/storage/redisstorage/session.go` - RefreshToken Set/Get/Delete
  - Gateway: Rate limiting for 2FA verification attempts

**File Storage:**
- Not used - All data persisted to PostgreSQL/Redis

**Caching:**
- Redis used for session storage (not caching)

## Authentication & Identity

**Auth Provider:**
- Custom internal system (no external provider)
- Implementation: `auth/internal/services/authService/`

**JWT Tokens:**
- Type: RS256 (RSA-2048 + SHA-256)
- Issuer: Auth Service
- Consumers: API Gateway, TwoFA Service, other services via ValidateToken RPC
- Token Details:
  - Access: 15 minutes (verified on each request)
  - Refresh: 7 days (stored in Redis, validated on refresh request)
  - Claims: `sub` (user_id UUID), `email`, `iat`, `exp`

**Password Security:**
- Algorithm: bcrypt cost=12
- Validation policy: 12+ characters, mixed case, digits, special chars, no 4+ character sequences
- Implementation: `auth/internal/services/authService/password_validation.go`

**Multi-Factor Authentication:**
- Type: TOTP (RFC 6238, 6-digit codes)
- Secret Distribution: Shamir Secret Sharing (2-of-3, custom GF(256) implementation)
- Storage: Only encrypted shares in MPC nodes (not centralized)

## Monitoring & Observability

**Metrics Collection:**
- System: Prometheus
- Client: github.com/prometheus/client_golang
- Metrics per service:
  - `auth_requests_total` (method, status)
  - `auth_request_duration_seconds`
  - `twofa_operations_total` (operation, status)
  - `twofa_mpc_latency_seconds` (node)
  - `mpc_operations_total` (node_id, operation, status)
  - `mpc_operation_duration_seconds`
- Scrape configuration: Not shown in codebase (expected in `monitoring/` directory)

**Logs:**
- Framework: slog (standard library structured logging)
- Format: JSON structured logs
- Destination: STDOUT (containers capture to ELK/CloudWatch/etc)
- Redaction: Never log passwords, TOTP secrets, share data, encryption keys

**Dashboards:**
- System: Grafana
- Configuration: `monitoring/` directory (not yet implemented in current codebase state)
- Expected dashboards: Auth metrics, 2FA verification rates, MPC node health, PostgreSQL/Redis connections

**Error Tracking:**
- Not configured (only gRPC error codes returned)

## CI/CD & Deployment

**Hosting:**
- Unspecified (can run locally or on Kubernetes)
- Docker Compose for local development (PostgreSQL, Redis, Kafka)

**CI Pipeline:**
- Not configured (no GitHub Actions, GitLab CI, etc. detected)

**Container Registry:**
- Not configured

## Environment Configuration

**Required Environment Variables (by Service):**

**All Services:**
```
# PostgreSQL
DATABASE_URL=postgresql://user:pass@localhost:5432/service_db

# Kafka
KAFKA_BROKERS=localhost:9092

# Server
PORT=50051 (or mapped from config.yaml)
```

**Auth Service:**
```
# Redis
REDIS_URL=redis://localhost:6379

# JWT (RSA keys)
JWT_PRIVATE_KEY_PATH=/path/to/private.pem
JWT_PUBLIC_KEY_PATH=/path/to/public.pem
```

**TwoFA Service:**
```
# MPC Nodes
MPC_NODE_1_ADDR=localhost:50052
MPC_NODE_2_ADDR=localhost:50053
MPC_NODE_3_ADDR=localhost:50054

# Timeouts
MPC_RPC_TIMEOUT_SECONDS=5
```

**MPC Node:**
```
# Encryption
ENCRYPTION_KEY=<base64-encoded 32-byte key>

# Node Identity
NODE_ID=1  # or 2, 3

# Service Identity (optional)
NODE_NAME=mpc-node-1
```

**Gateway:**
```
# Redis (rate limiting)
REDIS_URL=redis://localhost:6379

# Service Addresses
AUTH_SERVICE_ADDR=localhost:50051
TWOFA_SERVICE_ADDR=localhost:50051
```

**Secrets Location:**
- Configuration file: `config.yaml` per service
- Sensitive keys (encryption, JWT keys): Managed via config.yaml or environment variables
- Not stored in `.env` files in repository

## Webhooks & Callbacks

**Incoming:**
- None (no external webhooks received)

**Outgoing:**
- None (no external webhooks sent)

## Inter-Service Communication

**gRPC Endpoints:**

**Auth Service (Exposed):**
- `authapi.AuthService/Register` - User registration
- `authapi.AuthService/Login` - User login
- `authapi.AuthService/RefreshToken` - Token rotation
- `authapi.AuthService/Logout` - Session termination
- `authapi.AuthService/ValidateToken` - Token verification (used by Gateway)

**TwoFA Service (Exposed):**
- `twofaapi.TwoFAService/Setup2FA` - Provision TOTP
- `twofaapi.TwoFAService/Verify2FA` - Verify TOTP code
- `twofaapi.TwoFAService/Disable2FA` - Disable 2FA
- `twofaapi.TwoFAService/Get2FAStatus` - Check if 2FA enabled

**MPC Node Service (Exposed):**
- `mpcapi.MPCNodeService/StoreShare` - Store encrypted share fragment
- `mpcapi.MPCNodeService/RetrieveShare` - Retrieve encrypted share fragment
- `mpcapi.MPCNodeService/DeleteShare` - Delete all shares for user

**Gateway Clients (Internal):**
- Auth Service (synchronous RPC)
- TwoFA Service (synchronous RPC)
- Redis (for rate limiting)

**TwoFA Clients:**
- MPC Node 1, 2, 3 (parallel gRPC calls with 5-second timeout)

**MPC Node Clients:**
- PostgreSQL (share storage)
- Kafka (audit events)

## Message Queue

**Kafka:**
- Cluster: Single or replicated (configured in `config.yaml`)
- Topics (one per service or unified):
  - Auth Service produces:
    - `user.registered` - New user registration
    - `user.logged_in` - Successful login
    - `token.refreshed` - Token rotation event
  - TwoFA Service produces:
    - `2fa.verified` - 2FA code verified
    - `2fa.disabled` - 2FA disabled
  - MPC Node produces:
    - `share.stored` - Share encrypted and stored
    - `share.retrieved` - Share requested
    - `share.deleted` - User shares deleted

**Event Format:**
```json
{
  "user_id": "uuid",
  "operation": "string",
  "timestamp": "iso8601",
  "node_id": "1" // (mpc only)
}
```

**Security:** Never includes passwords, TOTP secrets, share data, encryption keys

**Consumers:** Not implemented in current codebase phase (events for future audit/compliance systems)

## Rate Limiting

**Implementation:**
- Redis-backed counters per `{user_id}:{operation}`
- Enforced in Gateway and TwoFA Service
- 2FA verification: 5 attempts per 5 minutes per user

**Storage:**
- Redis TTL: 5 minutes per counter

## Cross-Service Security

**Inter-Service Authentication:**
- Auth Service: ValidateToken RPC (JWT validation)
- MPC Nodes: Shared secret via gRPC metadata header (not yet fully specified in CLAUDE.md)

**Data Isolation:**
- Each service has separate PostgreSQL database (or schema)
- Kafka events stripped of sensitive data

---

*Integration audit: 2026-04-11*
