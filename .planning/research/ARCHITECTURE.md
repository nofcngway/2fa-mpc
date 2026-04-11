# Architecture Patterns

**Domain:** 2FA authentication system with MPC distributed secret storage
**Researched:** 2026-04-11

## Recommended Architecture

```
                          +------------------+
                          |   API Gateway    |  (out of scope now)
                          |  REST -> gRPC    |
                          +--------+---------+
                                   |
                    +--------------+--------------+
                    |                              |
             +------+------+              +-------+-------+
             | Auth Service |              | TwoFA Service |
             |  (gRPC)     |              |   (gRPC)      |
             +------+------+              +---+---+---+---+
                    |                         |   |   |
                    |                    +----+   |   +----+
              +-----+-----+             |        |        |
              |           |        +----+--+ +---+---+ +--+----+
           +--+--+   +----+--+     |MPC    | |MPC    | |MPC    |
           |Redis|   |Postgres|    |Node 1 | |Node 2 | |Node 3 |
           +-----+   +-------+    +---+----+ +---+---+ +---+---+
                                       |          |         |
                                   +---+---+ +---+---+ +---+---+
                                   |PG (1) | |PG (2) | |PG (3) |
                                   +-------+ +-------+ +-------+

All services --> Kafka (audit events)
All services --> Prometheus (metrics)
```

### Component Boundaries

| Component | Responsibility | Communicates With | Owns |
|-----------|---------------|-------------------|------|
| Auth Service | User registration, login, JWT issuance/validation, session management | Redis (sessions), PostgreSQL (users), Kafka (audit) | Users table, refresh tokens, RSA key pair |
| TwoFA Service | 2FA orchestration: Shamir split/combine, TOTP generation/validation, rate limiting | 3 MPC Nodes (gRPC), Redis (rate limits), PostgreSQL (2FA metadata), Kafka (audit) | 2FA metadata (enabled/disabled, backup codes), Shamir GF(256) implementation, TOTP logic |
| MPC Node (x3) | Store, retrieve, delete encrypted secret shares | PostgreSQL (shares), Kafka (audit) | One share per user, AES-256-GCM encryption key |
| Auth's PostgreSQL | Users, credentials | Auth Service only | users table |
| TwoFA's PostgreSQL | 2FA metadata, backup codes | TwoFA Service only | twofa_metadata, backup_codes tables |
| MPC Node's PostgreSQL | Encrypted shares | Respective MPC Node only | shares table |
| Redis | Refresh tokens (Auth), rate limit counters (TwoFA) | Auth Service, TwoFA Service | Session state, counters |
| Kafka | Audit event bus | All services produce; consumers out of scope | Audit topic(s) |

### Data Flow

#### Registration + Login Flow
```
Client -> Auth.Register(email, password)
  Auth: validate password -> bcrypt hash -> store in PG -> produce Kafka audit
  Auth: return success

Client -> Auth.Login(email, password)
  Auth: verify credentials -> generate JWT (RS256 access 15m + refresh 7d)
  Auth: store refresh token in Redis with TTL -> produce Kafka audit
  Auth: return {access_token, refresh_token}
```

#### 2FA Setup Flow (critical path)
```
Client -> TwoFA.Setup2FA(user_id)
  TwoFA: generate TOTP secret (20 bytes, RFC 6238)
  TwoFA: Shamir split secret into 3 shares (threshold=2) in GF(256)
  TwoFA: fan-out gRPC calls to 3 MPC nodes in parallel:
    MPC1.StoreShare(user_id, share_1)  -- MPC encrypts with AES-256-GCM, stores
    MPC2.StoreShare(user_id, share_2)
    MPC3.StoreShare(user_id, share_3)
  TwoFA: zeroize TOTP secret from memory (overwrite bytes)
  TwoFA: zeroize shares from memory
  TwoFA: store 2FA metadata (enabled=pending) in PG
  TwoFA: generate backup codes, bcrypt hash each, store hashes in PG
  TwoFA: produce Kafka audit event
  TwoFA: return {provisioning_uri, backup_codes_plaintext}
```

#### 2FA Verify Flow (critical path)
```
Client -> TwoFA.Verify2FA(user_id, otp_code)
  TwoFA: check rate limit in Redis (5 attempts / 5 min per user_id)
  TwoFA: fan-out gRPC calls to 2 of 3 MPC nodes:
    MPC1.RetrieveShare(user_id)  -- MPC decrypts, returns share
    MPC2.RetrieveShare(user_id)
  TwoFA: Shamir combine 2 shares -> reconstruct TOTP secret
  TwoFA: validate OTP against secret (current window +/- 1)
  TwoFA: zeroize TOTP secret from memory
  TwoFA: zeroize shares from memory
  TwoFA: produce Kafka audit event
  TwoFA: return {valid: true/false}
```

## Internal Service Structure (Clean Architecture per Service)

Each service follows the medialog/students pattern. The dependency flow is strictly inward:

```
Proto/gRPC Handler (api layer)
       |
       v  calls interface
Service Layer (business logic)
       |
       v  calls interface
Storage Layer (repository, pgx)
```

### Layer Responsibilities

| Layer | Package | Responsibility | Depends On |
|-------|---------|---------------|------------|
| Proto definitions | `api/` | .proto files defining RPC methods and models | Nothing |
| Generated code | `internal/pb/` | protoc-generated Go stubs | Proto definitions |
| gRPC Handlers | `internal/api/<service>_service_api/` | Translate gRPC request/response, call service interface, return gRPC status codes | Service interface |
| Service (business) | `internal/services/<name>/` | Business logic, orchestration, validation | Repository interface, external client interfaces |
| Storage (repository) | `internal/storage/pgstorage/` | SQL queries via pgx, data mapping | pgx pool (injected) |
| Models | `internal/models/` | Domain structs used across layers | Nothing |
| Bootstrap (DI) | `internal/bootstrap/` | Factory functions creating concrete types, wiring interfaces | All concrete types |
| Middleware | `internal/middleware/` | gRPC interceptors (logging, metrics, auth) | slog, prometheus |
| Config | `config/` | YAML config loading | gopkg.in/yaml.v3 |
| Entrypoint | `cmd/app/main.go` | Bootstrap, start server, graceful shutdown | Bootstrap factories |

### Bootstrap / DI Pattern (No Framework)

Each dependency gets a factory function in `internal/bootstrap/`. The composition root is `cmd/app/main.go` which calls factories in order.

```go
// internal/bootstrap/postgres.go
func NewPostgresPool(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
    pool, err := pgxpool.New(ctx, cfg.DSN)
    if err != nil {
        return nil, fmt.Errorf("bootstrap postgres: %w", err)
    }
    return pool, nil
}

// internal/bootstrap/storage.go
func NewUserStorage(pool *pgxpool.Pool) *pgstorage.UserStorage {
    return pgstorage.NewUserStorage(pool)
}

// internal/bootstrap/service.go
func NewAuthService(storage services.UserRepository, ...) *auth.Service {
    return auth.NewService(storage, ...)
}

// cmd/app/main.go (composition root)
func main() {
    cfg := config.MustLoad("config.yaml")
    
    pool, err := bootstrap.NewPostgresPool(ctx, cfg.Postgres)
    // ... error handling
    
    storage := bootstrap.NewUserStorage(pool)
    service := bootstrap.NewAuthService(storage, ...)
    handler := bootstrap.NewAuthHandler(service)
    
    grpcServer := grpc.NewServer(interceptors...)
    pb.RegisterAuthServiceServer(grpcServer, handler)
    
    // health + graceful shutdown...
}
```

Key principle: every dependency is created via a factory in bootstrap, injected through constructor parameters. No global state, no service locator, no reflection-based DI framework.

### One File Per Method Pattern

Each RPC method gets its own file in both the handler and service layers:

```
internal/api/auth_service_api/
    register.go        // Register RPC handler
    login.go           // Login RPC handler
    refresh_token.go   // RefreshToken RPC handler
    logout.go          // Logout RPC handler
    validate_token.go  // ValidateToken RPC handler

internal/services/auth/
    service.go         // Service struct + constructor + interface
    register.go        // Register business logic
    login.go           // Login business logic
    refresh_token.go   // RefreshToken business logic
    logout.go          // Logout business logic
    validate_token.go  // ValidateToken business logic
```

This avoids 1000-line god files and makes git diffs clean per feature.

## Patterns to Follow

### Pattern 1: Graceful Shutdown with Multiple Connections

Every service manages PostgreSQL pool, gRPC server, Redis client, Kafka writer, and Prometheus HTTP server. Shutdown must be ordered.

```go
// cmd/app/main.go
func main() {
    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

    // ... bootstrap all dependencies ...

    // Start gRPC server in goroutine
    go func() {
        if err := grpcServer.Serve(lis); err != nil {
            slog.Error("gRPC server failed", "error", err)
        }
    }()

    slog.Info("service started", "addr", cfg.GRPC.Addr)

    // Block until signal
    <-ctx.Done()
    slog.Info("shutting down...")

    // 1. Stop accepting new RPCs, drain in-flight
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer shutdownCancel()

    stopped := make(chan struct{})
    go func() {
        grpcServer.GracefulStop()
        close(stopped)
    }()

    select {
    case <-stopped:
        slog.Info("gRPC server stopped gracefully")
    case <-shutdownCtx.Done():
        slog.Warn("gRPC graceful stop timed out, forcing")
        grpcServer.Stop()
    }

    // 2. Close Kafka writer (flush pending messages)
    if err := kafkaWriter.Close(); err != nil {
        slog.Error("kafka writer close failed", "error", err)
    }

    // 3. Close Redis
    if err := redisClient.Close(); err != nil {
        slog.Error("redis close failed", "error", err)
    }

    // 4. Close PostgreSQL pool (last, after all queries done)
    pgPool.Close()

    slog.Info("shutdown complete")
}
```

Shutdown order rationale: gRPC first (stop accepting work) -> Kafka (flush audit) -> Redis (release connections) -> PostgreSQL (last, ensures all pending queries finish).

### Pattern 2: gRPC Health Check

Use the built-in `google.golang.org/grpc/health` package. No custom proto needed.

```go
import (
    "google.golang.org/grpc/health"
    healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

healthServer := health.NewServer()
healthpb.RegisterHealthServer(grpcServer, healthServer)

// Set status for the overall server
healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

// Set status for specific service (optional, useful for k8s probes)
healthServer.SetServingStatus("auth.AuthService", healthpb.HealthCheckResponse_SERVING)

// During shutdown:
healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
```

### Pattern 3: Fan-Out gRPC Calls (TwoFA -> MPC Nodes)

TwoFA must call 2 or 3 MPC nodes in parallel. Use errgroup for coordinated fan-out with context cancellation.

```go
import "golang.org/x/sync/errgroup"

func (s *TwoFAService) storeShares(ctx context.Context, userID string, shares [][]byte) error {
    g, ctx := errgroup.WithContext(ctx)

    for i, client := range s.mpcClients {
        i, client := i, client // capture loop vars
        g.Go(func() error {
            _, err := client.StoreShare(ctx, &mpcpb.StoreShareRequest{
                UserId:   userID,
                ShareData: shares[i],
                ShareIndex: int32(i + 1),
            })
            return err
        })
    }

    if err := g.Wait(); err != nil {
        return fmt.Errorf("store shares: %w", err)
    }
    return nil
}
```

For retrieval (only need 2 of 3), use a different pattern -- launch all 3, take first 2 successes:

```go
func (s *TwoFAService) retrieveShares(ctx context.Context, userID string) ([]Share, error) {
    type result struct {
        share Share
        err   error
    }

    results := make(chan result, len(s.mpcClients))

    for i, client := range s.mpcClients {
        go func(idx int, c mpcpb.MPCNodeServiceClient) {
            resp, err := c.RetrieveShare(ctx, &mpcpb.RetrieveShareRequest{
                UserId: userID,
            })
            if err != nil {
                results <- result{err: err}
                return
            }
            results <- result{share: Share{
                Index: int(resp.ShareIndex),
                Data:  resp.ShareData,
            }}
        }(i, client)
    }

    var shares []Share
    var errs []error
    for range s.mpcClients {
        r := <-results
        if r.err != nil {
            errs = append(errs, r.err)
            continue
        }
        shares = append(shares, r.share)
        if len(shares) >= 2 { // threshold met
            return shares, nil
        }
    }

    return nil, fmt.Errorf("need 2 shares, got %d (errors: %v)", len(shares), errs)
}
```

This pattern provides fault tolerance: if 1 MPC node is down, the system still works.

### Pattern 4: gRPC Interceptors (Middleware Chain)

Each service needs logging, metrics, and optionally auth interceptors.

```go
grpcServer := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        middleware.LoggingInterceptor(logger),
        middleware.MetricsInterceptor(promRegistry),
        middleware.RecoveryInterceptor(),
    ),
)
```

For MPC nodes, add an auth interceptor that checks shared secret in gRPC metadata:

```go
func SharedSecretInterceptor(secret string) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
        md, ok := metadata.FromIncomingContext(ctx)
        if !ok {
            return nil, status.Error(codes.Unauthenticated, "missing metadata")
        }
        tokens := md.Get("authorization")
        if len(tokens) == 0 || tokens[0] != secret {
            return nil, status.Error(codes.Unauthenticated, "invalid shared secret")
        }
        return handler(ctx, req)
    }
}
```

### Pattern 5: Kafka Audit Producer (Async, Non-Blocking)

Audit events must never block the main request path. Use segmentio/kafka-go with async writes.

```go
// internal/audit/producer.go
type AuditProducer struct {
    writer *kafka.Writer
}

type AuditEvent struct {
    UserID    string    `json:"user_id"`
    Operation string    `json:"operation"`
    ServiceName string  `json:"service_name"`
    Timestamp time.Time `json:"timestamp"`
    Success   bool      `json:"success"`
    // NEVER include secret data, shares, passwords, keys
}

func (p *AuditProducer) Emit(ctx context.Context, event AuditEvent) {
    data, err := json.Marshal(event)
    if err != nil {
        slog.Error("audit marshal failed", "error", err)
        return
    }

    // Fire-and-forget: audit should not block business logic
    go func() {
        err := p.writer.WriteMessages(context.Background(), kafka.Message{
            Key:   []byte(event.UserID),
            Value: data,
        })
        if err != nil {
            slog.Error("audit write failed", "error", err)
        }
    }()
}
```

### Pattern 6: Secret Zeroization

TOTP secrets and shares must be wiped from memory after use.

```go
func zeroize(b []byte) {
    for i := range b {
        b[i] = 0
    }
}

// Usage in TwoFA service:
secret := generateTOTPSecret()
defer zeroize(secret)

shares := shamirSplit(secret, 3, 2)
defer func() {
    for _, s := range shares {
        zeroize(s)
    }
}()
// ... store shares to MPC nodes ...
// secret and shares are zeroed when function returns
```

## Proto File Organization

### Strategy: Each Service Owns Its Proto, Consumer Copies

Given that services are separate Go modules (no shared monorepo module), the cleanest approach for this project:

1. **Auth** defines its proto in `auth/api/auth_api/auth.proto`
2. **TwoFA** defines its proto in `twofa/api/twofa_api/twofa.proto`
3. **TwoFA** also defines the MPC contract in `twofa/api/mpc_api/mpc.proto` (TwoFA is the consumer)
4. **MPC** copies `mpc.proto` from TwoFA into `mpc/api/mpc_api/mpc.proto`

Proto models shared across services go in `<service>/api/models/`. Each service generates its own Go stubs into `internal/pb/`.

```
auth/
  api/
    models/user.proto           # User model
    auth_api/auth.proto         # Auth RPC service
  internal/pb/                  # Generated Go code

twofa/
  api/
    models/twofa.proto          # 2FA metadata model
    twofa_api/twofa.proto       # TwoFA RPC service
    mpc_api/mpc.proto           # MPC contract (TwoFA defines it)
  internal/pb/                  # Generated Go code (includes MPC client stubs)

mpc/
  api/
    models/share.proto          # Share model
    mpc_api/mpc.proto           # Copied from twofa/api/mpc_api/
  internal/pb/                  # Generated Go code (MPC server stubs)
```

**Why TwoFA defines MPC proto**: TwoFA is the sole consumer of MPC nodes. The contract is driven by what TwoFA needs. MPC implements it.

## Anti-Patterns to Avoid

### Anti-Pattern 1: Storing TOTP Secret Anywhere Persistently
**What:** Writing the full TOTP secret to any database, file, or log.
**Why bad:** Defeats the entire purpose of Shamir splitting and MPC distribution.
**Instead:** Secret exists only transiently in TwoFA service memory during setup and verify. Zeroize immediately after use.

### Anti-Pattern 2: Synchronous MPC Calls (Sequential, Not Parallel)
**What:** Calling MPC nodes one-by-one, waiting for each response.
**Why bad:** Triples latency on the critical 2FA verify path. If one node is slow, everything is slow.
**Instead:** Fan-out with goroutines, collect 2-of-3 results. See Pattern 3 above.

### Anti-Pattern 3: God File Services
**What:** Putting all RPC handler methods or all business logic methods in a single file.
**Why bad:** Files grow to 1000+ lines. Git merge conflicts. Hard to navigate.
**Instead:** One file per method in both handler and service layers.

### Anti-Pattern 4: Leaking Domain Models Into Proto
**What:** Using protobuf-generated structs as domain models throughout the service.
**Why bad:** Couples business logic to wire format. Changes to proto break service internals.
**Instead:** Define domain models in `internal/models/`. Map between proto types and domain types in the handler layer.

### Anti-Pattern 5: Blocking on Audit Writes
**What:** Making Kafka audit writes synchronous in the request path.
**Why bad:** If Kafka is slow or down, all requests slow down or fail.
**Instead:** Fire-and-forget async writes. Log errors but do not fail the request.

### Anti-Pattern 6: DI Framework / Reflection Magic
**What:** Using Wire, Uber Fx, or dig for dependency injection in a small service.
**Why bad:** Adds complexity, hides wiring, makes debugging harder. These services have ~5-8 dependencies each.
**Instead:** Manual bootstrap factories. Explicit, readable, debuggable.

## Build Order and Dependencies

### Phase 1: Auth Service
**Why first:** No dependencies on other services. Self-contained. Establishes patterns (project structure, bootstrap, config loading, graceful shutdown, health check, Kafka audit, Prometheus metrics) that TwoFA and MPC will copy.

Build order within Auth:
1. Proto definitions + code generation pipeline (Makefile, generate.sh)
2. Config loading (config.yaml + config.go)
3. PostgreSQL storage layer (pgx, initTables, user CRUD)
4. Redis integration (refresh token store)
5. Service layer (register, login, refresh, logout, validate)
6. gRPC handlers
7. Bootstrap factories + main.go (with health check, graceful shutdown)
8. Interceptors (logging, metrics, recovery)
9. Kafka audit producer
10. Docker Compose (PostgreSQL, Redis, Kafka)

### Phase 2: MPC Node Service
**Why second:** Simpler than TwoFA. Implements a straightforward store/retrieve/delete API. Proto is defined by TwoFA but the MPC implementation is independent.

Build order within MPC:
1. Proto definitions (copy from TwoFA's mpc_api/)
2. Config + bootstrap skeleton (reuse patterns from Auth)
3. AES-256-GCM encryption/decryption module
4. PostgreSQL storage (encrypted shares)
5. Service layer (store, retrieve, delete)
6. gRPC handlers + shared secret auth interceptor
7. main.go, health check, graceful shutdown
8. Kafka audit + metrics

### Phase 3: TwoFA Service
**Why last:** Depends on MPC nodes being available to test. Contains the most complex logic (Shamir GF(256), TOTP, fan-out to MPC nodes, rate limiting). Building it last means all patterns are established and MPC nodes can be tested against.

Build order within TwoFA:
1. Proto definitions (twofa_api/ + mpc_api/ for client stubs)
2. Shamir Secret Sharing in GF(256) -- pure math, no dependencies, testable in isolation
3. TOTP implementation (RFC 6238) -- also pure math, testable in isolation
4. Config + bootstrap skeleton
5. gRPC client connections to MPC nodes (bootstrap factory)
6. PostgreSQL storage (2FA metadata, backup codes)
7. Redis integration (rate limiting)
8. Service layer (setup, verify, disable, status) with fan-out pattern
9. gRPC handlers
10. main.go, health check, graceful shutdown
11. Kafka audit + metrics

### Dependency Graph

```
Auth Service (independent, build first)
    |
    | (establishes patterns)
    v
MPC Node Service (independent of Auth, but uses same patterns)
    |
    | (must be running for TwoFA integration tests)
    v
TwoFA Service (depends on MPC Node proto contract + running MPC nodes for testing)
```

Note: Auth and MPC could technically be built in parallel since they are independent. But building Auth first establishes all the boilerplate patterns (config, bootstrap, shutdown, health, interceptors, Kafka, Prometheus) that MPC and TwoFA will reuse.

## Scalability Considerations

| Concern | Current ( demo) | Production-ready |
|---------|---------------------|------------------|
| MPC nodes | 3 nodes, same machine (docker-compose) | 3+ nodes on separate physical machines for actual security |
| PostgreSQL | One instance, multiple databases | Separate instances per service |
| Redis | Single instance | Redis Sentinel or Cluster |
| Kafka | Single broker | Multi-broker cluster with replication |
| Auth token validation | Direct gRPC call to Auth | JWT public key distribution (Gateway validates locally) |
| TwoFA rate limiting | Redis counter per user | Redis + sliding window algorithm |

For the  scope, single-machine docker-compose is sufficient. The architecture supports scaling each component independently later.

## Sources

- [gRPC Graceful Shutdown Guide](https://grpc.io/docs/guides/server-graceful-stop/) -- official gRPC documentation
- [Go Graceful Shutdown Practical Patterns](https://victoriametrics.com/blog/go-graceful-shutdown/) -- VictoriaMetrics
- [grpc-go Concurrency Documentation](https://github.com/grpc/grpc-go/blob/master/Documentation/concurrency.md) -- official concurrency safety notes
- [gRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md) -- official protocol spec
- [google.golang.org/grpc/health package](https://pkg.go.dev/google.golang.org/grpc/health) -- Go health check implementation
- [Sharing gRPC Protobufs Between Microservices](https://jozefcipa.com/blog/sharing-grpc-protobufs-between-microservices/) -- proto organization strategies
- [Go Project Structure: Clean Architecture Patterns](https://dasroot.net/posts/2026/01/go-project-structure-clean-architecture/) -- 2026 patterns
- [Three Dots Labs: Clean Architecture in Go](https://threedots.tech/post/introducing-clean-architecture/) -- authoritative Go clean arch guide
- [DI Without Frameworks in Go](https://oneuptime.com/blog/post/2026-01-30-how-to-implement-dependency-injection-without-frameworks-in-go/view) -- manual DI patterns
- [Go Microservices in 2025](https://medium.com/@QuarkAndCode/go-microservices-in-2025-architecture-grpc-vs-rest-frameworks-09159c95a8d0) -- architecture decisions
- [Fan-Out Fan-In in Go](https://dev.to/silver_dev/concurrency-patterns-on-golang-fan-out-fan-in-goj) -- concurrency patterns
