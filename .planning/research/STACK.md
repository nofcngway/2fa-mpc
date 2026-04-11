# Technology Stack

**Project:** MPC-2FA (Distributed 2FA with Shamir Secret Sharing)
**Researched:** 2026-04-11
**Overall confidence:** HIGH — Stack is pre-decided, research validates versions and best practices

## Recommended Stack

### Core Language & Runtime

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| Go | 1.26.2 | Service language | Pre-decided. Excellent for microservices: goroutines, strong stdlib, crypto packages, slog built-in | HIGH |

### gRPC & Protobuf

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| google.golang.org/grpc | ~v1.79 | gRPC framework | The standard Go gRPC implementation. Active development (Feb 2026 release) | HIGH |
| google.golang.org/protobuf | latest | Protobuf runtime | Official Go protobuf runtime from google.golang.org (NOT github.com/golang/protobuf which is legacy) | HIGH |
| protoc-gen-go | latest | Proto message codegen | Generates Go structs from .proto files | HIGH |
| protoc-gen-go-grpc | latest | Proto service codegen | Generates gRPC server/client stubs | HIGH |
| grpc-ecosystem/go-grpc-middleware/v2 | v2.2.0 | Interceptor toolkit | Provides logging (slog adapter), recovery, auth, rate limiting, selector interceptors. Replaces writing boilerplate interceptors from scratch | HIGH |
| google.golang.org/grpc/health | (bundled with grpc) | Health checking | Built-in gRPC health check protocol implementation — Check() and Watch() methods. Required per project spec | HIGH |

**Protobuf generation approach:** Use `protoc` with `protoc-gen-go` and `protoc-gen-go-grpc` directly via a `generate.sh` script per service. Buf is superior for large teams but adds unnecessary complexity for a 4-service academic project. Stick with protoc + shell script as the reference project (medialog/students) uses.

**Proto generation command pattern:**
```bash
protoc \
  --go_out=./internal/pb --go_opt=paths=source_relative \
  --go-grpc_out=./internal/pb --go-grpc_opt=paths=source_relative \
  -I ./api \
  ./api/<service>_api/*.proto ./api/models/*.proto
```

### Database

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| PostgreSQL | latest stable | Persistent storage | Pre-decided. Users, sessions, 2FA metadata, encrypted shares | HIGH |
| github.com/jackc/pgx/v5 | v5.8.x | PostgreSQL driver | Pre-decided. Best Go PG driver: native protocol, connection pooling via pgxpool, no ORM needed. Active (Mar 2026 release) | HIGH |

**pgx best practices:**
- Use `pgxpool.New()` or `pgxpool.NewWithConfig()` — NEVER raw `pgx.Conn` for concurrent access (not thread-safe)
- Config must be created via `pgxpool.ParseConfig()` — manual struct construction panics
- Set pool limits: `MaxConns` (default: 4, set to ~10-25 per service), `MinConns` (2-5), `MaxConnLifetime` (1h), `MaxConnIdleTime` (30m)
- Use `pool.QueryRow()` / `pool.Query()` / `pool.Exec()` directly — pool manages connection acquisition
- Table initialization via `initTables` function that runs CREATE TABLE IF NOT EXISTS on startup (per project convention)

### Caching & Sessions

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| Redis | 8.6.2 | Sessions, rate limiting | Pre-decided. Refresh tokens with TTL, rate limit counters for 2FA verification | HIGH |
| github.com/redis/go-redis/v9 | v9.18.x | Redis client | Pre-decided. Official Redis Go client, supports Redis 8.6, connection pooling, pipelining | HIGH |

**Redis session patterns for this project:**
- Refresh tokens: `SET refresh:<token_hash> <user_id> EX 604800` (7 days TTL)
- Rate limiting: `INCR ratelimit:2fa:<user_id>` with `EXPIRE ratelimit:2fa:<user_id> 300` (5 min window)
- Use `redis.NewClient()` with `Options{Addr, Password, DB, PoolSize, MinIdleConns}`
- PoolSize: 10 per service (default), MinIdleConns: 5

### Event Streaming

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| Kafka | 4.1.2 | Audit events | Pre-decided. Async audit trail between services | HIGH |
| github.com/segmentio/kafka-go | v0.4.x | Kafka client | Pre-decided. Pure Go, no CGO dependency (unlike confluent-kafka-go), simple Writer/Reader API | HIGH |

**kafka-go Writer best practices:**
- Use `kafka.Writer` (high-level API) — handles retries, reconnections, batching automatically
- Set `RequiredAcks: kafka.RequireAll` for audit reliability (all replicas must acknowledge)
- Set `Async: false` for audit events — synchronous writes ensure no audit loss
- Use `Balancer: &kafka.LeastBytes{}` for even distribution
- Call `writer.Close()` in graceful shutdown to flush pending messages
- Topic: `audit-events` with JSON-encoded messages containing `{user_id, operation, timestamp, service, metadata}`
- NEVER include secrets, shares, or keys in audit messages

### Authentication & Cryptography

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| github.com/golang-jwt/jwt/v5 | v5.3.x | JWT tokens | Pre-decided. RS256 signing/verification with RSA key pairs. Production-ready, actively maintained | HIGH |
| golang.org/x/crypto | latest | bcrypt | Pre-decided. `golang.org/x/crypto/bcrypt` for password hashing at cost=12 | HIGH |
| crypto/aes + crypto/cipher | stdlib | AES-256-GCM | Go stdlib. No external dependency needed for authenticated encryption | HIGH |
| crypto/rand | stdlib | Secure randomness | Nonce generation for AES-GCM, TOTP secret generation | HIGH |

**JWT RS256 patterns:**
- Generate RSA key pair offline: `openssl genrsa -out private.pem 2048` / `openssl rsa -in private.pem -pubout -out public.pem`
- Load keys via `jwt.ParseRSAPrivateKeyFromPEM()` and `jwt.ParseRSAPublicKeyFromPEM()`
- Auth service signs with private key; Gateway/other services verify with public key only
- Always validate algorithm in parser: `jwt.WithValidMethods([]string{"RS256"})`
- Access token: 15 min expiry, claims: `{sub: user_id, exp, iat, jti}`
- Refresh token: 7 day expiry, stored as hash in Redis

**AES-256-GCM patterns:**
- Key: 32 bytes from config/env, NEVER hardcoded
- `aes.NewCipher(key)` then `cipher.NewGCM(block)`
- Nonce: 12 bytes from `crypto/rand.Read()` — unique per encryption operation
- Prepend nonce to ciphertext: `nonce + gcm.Seal(nil, nonce, plaintext, nil)`
- Decrypt: split first 12 bytes as nonce, rest as ciphertext
- CRITICAL: Never reuse nonces with the same key (max 2^32 random nonces per key)

**bcrypt patterns:**
- `bcrypt.GenerateFromPassword([]byte(password), 12)` — cost=12 per spec
- `bcrypt.CompareHashAndPassword(hash, []byte(password))` for verification
- Validate password BEFORE hashing (12+ chars, mixed case, digit, special char, no sequences)

### Observability

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| github.com/prometheus/client_golang | v1.23.x | Metrics | Pre-decided. Standard Prometheus client for Go. Min Go 1.23 | HIGH |
| go-grpc-middleware/providers/prometheus | (part of middleware v2) | gRPC metrics | Replaces deprecated go-grpc-prometheus. Provides grpc_server_* and grpc_client_* metrics as interceptors | HIGH |
| log/slog | stdlib | Structured logging | Go stdlib since 1.21. JSON output, structured fields, no external dependency | HIGH |
| go-grpc-middleware/interceptors/logging | (part of middleware v2) | gRPC logging | slog adapter for gRPC call logging. Copy the slog example from the repo, don't import | HIGH |

**Prometheus metrics pattern for gRPC:**
```go
// Use go-grpc-middleware prometheus provider (NOT deprecated go-grpc-prometheus)
import "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"

srvMetrics := prometheus.NewServerMetrics()
srv := grpc.NewServer(
    grpc.ChainUnaryInterceptor(srvMetrics.UnaryServerInterceptor()),
    grpc.ChainStreamInterceptor(srvMetrics.StreamServerInterceptor()),
)
srvMetrics.InitializeMetrics(srv)
// Expose /metrics endpoint via http.Handle("/metrics", promhttp.Handler())
```

**slog pattern:**
```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
slog.SetDefault(logger)
// Use: slog.Info("user registered", "user_id", userID, "operation", "register")
// NEVER: slog.Info("...", "password", pw, "secret", secret, "share", share)
```

### Configuration & Utilities

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| gopkg.in/yaml.v3 | v3 | Config loading | Pre-decided. Loads config.yaml per service | HIGH |
| github.com/google/uuid | v1.6.0 | UUID generation | Pre-decided. UUIDv4 for user_id, session_id, share_id. RFC 9562 compliant | HIGH |

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| Protobuf gen | protoc + scripts | Buf | Overkill for 4-service academic project. protoc + generate.sh matches reference project pattern |
| Kafka client | segmentio/kafka-go | confluent-kafka-go | confluent requires CGO (librdkafka), complicates builds. kafka-go is pure Go |
| Kafka client | segmentio/kafka-go | IBM/sarama | sarama has more complex API, kafka-go is simpler for producer-only audit use case |
| PostgreSQL | pgx/v5 raw | GORM, sqlx, sqlc | ORM explicitly forbidden. pgx raw is the project requirement. sqlc would add codegen step |
| Logging | slog (stdlib) | zap, zerolog | slog is stdlib since Go 1.21, no external dependency, structured JSON output, sufficient for this project |
| JWT | golang-jwt/v5 | lestrrat-go/jwx | golang-jwt is simpler, widely adopted, RS256 works out of the box |
| gRPC metrics | go-grpc-middleware/prometheus | go-grpc-prometheus | go-grpc-prometheus is deprecated, replaced by the middleware provider |
| Config | yaml.v3 manual | viper, envconfig | yaml.v3 matches reference project. Viper adds unnecessary complexity for simple config.yaml loading |
| UUID | google/uuid | gofrs/uuid | google/uuid is sufficient, widely used, supports v4 and v7 |
| Shamir SSS | Custom GF(256) | hashicorp/vault SSS | Explicitly forbidden — must implement from scratch per academic requirements |
| TOTP | Custom RFC 6238 | pquerna/otp | Academic project — implement from scratch to demonstrate understanding |

## What NOT to Use

| Technology | Why Not |
|------------|---------|
| GORM / any ORM | Explicitly forbidden. Use pgx raw queries |
| Any Shamir library | Must implement from scratch in GF(256) |
| Any TOTP library | Academic project — implement RFC 6238 from scratch |
| github.com/golang/protobuf | Legacy. Use google.golang.org/protobuf |
| go-grpc-prometheus (standalone) | Deprecated. Use go-grpc-middleware/providers/prometheus |
| HTTP frameworks (gin, echo, chi) in services | Only gRPC. HTTP only in future Gateway |
| Viper | Overengineered for config.yaml loading. yaml.v3 is sufficient |
| mTLS between services | Project uses shared secret in gRPC metadata for MPC auth (simpler for academic scope) |

## Installation

```bash
# Core dependencies (per service go.mod)
go get google.golang.org/grpc
go get google.golang.org/protobuf
go get github.com/jackc/pgx/v5
go get github.com/redis/go-redis/v9
go get github.com/segmentio/kafka-go
go get github.com/golang-jwt/jwt/v5
go get github.com/prometheus/client_golang
go get github.com/google/uuid
go get golang.org/x/crypto
go get gopkg.in/yaml.v3

# Middleware (for interceptors)
go get github.com/grpc-ecosystem/go-grpc-middleware/v2

# Dev tools (installed globally, not in go.mod)
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

## Per-Service Dependency Matrix

| Dependency | Auth | TwoFA | MPC |
|------------|------|-------|-----|
| grpc | YES | YES | YES |
| protobuf | YES | YES | YES |
| pgx/v5 | YES | YES | YES |
| go-redis/v9 | YES | NO | NO |
| kafka-go | YES | YES | YES |
| golang-jwt/v5 | YES | NO | NO |
| x/crypto (bcrypt) | YES | NO | NO |
| crypto/aes+cipher | NO | NO | YES |
| prometheus/client_golang | YES | YES | YES |
| go-grpc-middleware/v2 | YES | YES | YES |
| google/uuid | YES | YES | YES |
| yaml.v3 | YES | YES | YES |

Notes:
- Redis is only needed in Auth (refresh tokens, sessions) and potentially Gateway (rate limiting, deferred)
- AES-256-GCM is only in MPC nodes (at-rest encryption of shares)
- JWT signing is only in Auth; verification will be in Gateway (deferred)
- TwoFA may need Redis for rate limiting (5 attempts/5min) — evaluate during implementation

## gRPC Server Bootstrap Pattern

Every service follows this pattern for server setup:

```go
// cmd/app/main.go
func main() {
    // 1. Load config
    cfg := config.MustLoad("config.yaml")

    // 2. Setup logger
    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))
    slog.SetDefault(logger)

    // 3. Bootstrap dependencies (DI)
    deps := bootstrap.NewDependencies(cfg)
    defer deps.Close()

    // 4. Create gRPC server with interceptors
    srvMetrics := prometheus.NewServerMetrics()
    srv := grpc.NewServer(
        grpc.ChainUnaryInterceptor(
            srvMetrics.UnaryServerInterceptor(),
            recovery.UnaryServerInterceptor(),
            // logging interceptor (slog adapter)
            // auth interceptor (where needed)
        ),
    )

    // 5. Register services
    pb.RegisterAuthServiceServer(srv, deps.AuthAPI)
    health.RegisterHealthServer(srv, deps.HealthServer)
    srvMetrics.InitializeMetrics(srv)

    // 6. Start metrics HTTP server (separate port)
    go func() {
        http.Handle("/metrics", promhttp.Handler())
        http.ListenAndServe(cfg.MetricsAddr, nil)
    }()

    // 7. Start gRPC server
    lis, _ := net.Listen("tcp", cfg.GRPCAddr)
    go srv.Serve(lis)

    // 8. Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    slog.Info("shutting down")
    srv.GracefulStop()
}
```

## Sources

- [grpc-go releases](https://github.com/grpc/grpc-go/releases) — v1.79.x, Feb 2026
- [pgx GitHub](https://github.com/jackc/pgx) — v5.8.x, Mar 2026
- [pgxpool docs](https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool) — connection pooling API
- [go-redis GitHub](https://github.com/redis/go-redis) — v9.18.x, supports Redis 8.6
- [segmentio/kafka-go](https://github.com/segmentio/kafka-go) — pure Go Kafka client
- [golang-jwt/jwt](https://github.com/golang-jwt/jwt) — v5.3.x, RS256 support
- [prometheus/client_golang](https://github.com/prometheus/client_golang) — v1.23.x
- [go-grpc-middleware](https://github.com/grpc-ecosystem/go-grpc-middleware) — v2.2.0, logging/metrics/recovery
- [grpc health check protocol](https://pkg.go.dev/google.golang.org/grpc/health) — built-in health checking
- [google/uuid](https://github.com/google/uuid) — v1.6.0, RFC 9562
- [AES-256-GCM in Go](https://gist.github.com/kkirsche/e28da6754c39d5e7ea10) — encryption pattern reference
- [protoc-gen-go](https://pkg.go.dev/google.golang.org/protobuf/cmd/protoc-gen-go) — official protobuf codegen
