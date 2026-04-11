# Phase 1: Project Scaffolding - Research

**Researched:** 2026-04-11
**Domain:** Go microservices scaffolding, gRPC, protobuf, Clean Architecture, Docker Compose
**Confidence:** HIGH

## Summary

Phase 1 creates runnable skeletons for three Go microservices (auth, twofa, mpc) following Clean Architecture with full proto definitions, config loading, Docker Compose infrastructure, and bootstrap DI wiring. All service directories currently exist but are empty.

The environment is fully prepared: Go 1.26.2, protoc 34.1, protoc-gen-go v1.36.11, protoc-gen-go-grpc 1.6.0, Docker 29.3.1, Docker Compose v5.1.1 are all installed. All Go dependency versions have been verified against the Go module proxy. The project uses well-documented patterns from CLAUDE.md and ADR-005.

**Primary recommendation:** Scaffold each service as a separate Go module following the exact directory structure from CLAUDE.md, with full proto RPC definitions (stub implementations returning `codes.Unimplemented`), config.yaml with all sections, and per-service docker-compose.yaml for local infrastructure.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Define full RPC methods and message types for each service from the start (per CLAUDE.md spec), but handler implementations return `codes.Unimplemented` -- stubs that compile and register correctly
- **D-02:** Proto models match the domain models described in workspace docs (User, TokenPair, Share, etc.)
- **D-03:** One docker-compose.yaml per service (per CLAUDE.md: "Docker Compose per service for local dependencies") -- each contains PostgreSQL and Redis as needed
- **D-04:** Kafka included in docker-compose where needed (auth, twofa, mpc all publish audit events per requirements) -- single shared Kafka instance referenced across services is acceptable for local dev
- **D-05:** MPC node uses a single docker-compose with one PostgreSQL instance -- 3 separate node instances are configured via different config.yaml files (different ports, node IDs, encryption keys)
- **D-06:** Include all config sections from the start (server, database, redis, kafka, jwt/encryption as applicable) with sensible local defaults -- avoids config refactoring in later phases
- **D-07:** RSA key paths in auth config.yaml point to `keys/` directory within auth service -- keys generated manually or via Makefile target, NOT committed to repo
- **D-08:** Use classic `protoc` with `protoc-gen-go` and `protoc-gen-go-grpc` -- simpler setup, no buf dependency, matches academic project scope
- **D-09:** Each service has its own `scripts/generate.sh` that generates from local `api/` directory into `internal/pb/`
- **D-10:** Bootstrap layer creates real dependencies (PGStorage, Redis, Kafka) but services can start even if some are unavailable -- log warnings, don't panic on optional deps like Kafka
- **D-11:** Interfaces defined in service files, implementations in storage -- standard Go Clean Architecture pattern per ADR-005

### Claude's Discretion
- Proto field types, naming, and package structure
- Makefile targets and scripts organization
- Exact docker-compose service names and port mappings
- initTables SQL schema details (minimal for skeleton phase)

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| INFRA-01 | Each service follows Clean Architecture -- handler -> service -> repository, dependencies via interfaces | Directory structure template from CLAUDE.md, ADR-005, bootstrap DI pattern |
| INFRA-02 | DI through bootstrap factories in internal/bootstrap/ | Bootstrap factory pattern with interface-based wiring |
| INFRA-08 | Configuration via config.yaml loaded in config/config.go | gopkg.in/yaml.v3 config loading pattern, full config struct per service |
| INFRA-09 | Proto definitions in api/ with generate.sh for protobuf code generation | protoc + protoc-gen-go + protoc-gen-go-grpc toolchain, generate.sh scripts |
| INFRA-10 | Each service is separate Go module (github.com/vbncursed/vkr/{auth,twofa,mpc}) | Separate go.mod per service directory |
| INFRA-11 | Docker Compose per service for local dependencies (PostgreSQL, Redis) | docker-compose.yaml per service with PostgreSQL 17 and Redis 8 images |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

- **No ORM**: pgx only, no GORM
- **No third-party Shamir libraries**: implement from scratch (Phase 4, not this phase)
- **No HTTP in services**: gRPC only (HTTP only in future Gateway)
- **Clean Architecture**: handler -> service -> repository, DI via bootstrap
- **Logging**: slog structured, NEVER log secrets/passwords/shares/keys
- **gRPC error codes**: InvalidArgument, NotFound, Unauthenticated, AlreadyExists, Internal
- **Config**: config.yaml loaded via config/config.go using gopkg.in/yaml.v3
- **Go modules**: `github.com/vbncursed/vkr/{auth,twofa,mpc}`
- **Service structure**: Must match CLAUDE.md template exactly (api/, cmd/app/, config/, internal/{api,bootstrap,models,services,storage,pb,middleware})
- **Naming**: snake_case files, PascalCase exports, SCREAMING_SNAKE for constants
- **gRPC Health Check Protocol** in each service
- **Graceful shutdown** with connection closure

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| google.golang.org/grpc | v1.80.0 | gRPC framework | Standard Go gRPC implementation [VERIFIED: go module proxy] |
| google.golang.org/protobuf | v1.36.11 | Protobuf runtime | Required for generated proto code [VERIFIED: go module proxy] |
| github.com/jackc/pgx/v5 | v5.9.1 | PostgreSQL driver | High-performance, pgx pool, no ORM [VERIFIED: go module proxy] |
| github.com/redis/go-redis/v9 | v9.18.0 | Redis client | Session storage, rate limiting [VERIFIED: go module proxy] |
| github.com/segmentio/kafka-go | v0.4.50 | Kafka client | Audit event publishing [VERIFIED: go module proxy] |
| gopkg.in/yaml.v3 | v3.0.1 | YAML config | Config file parsing [VERIFIED: go module proxy] |
| github.com/google/uuid | v1.6.0 | UUID generation | User/session IDs [VERIFIED: go module proxy] |

### Phase 1 Only (skeleton -- not yet needed in code but in go.mod)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/golang-jwt/jwt/v5 | v5.3.1 | JWT tokens | Phase 2-3 (Auth), but config references key paths now [VERIFIED: go module proxy] |
| golang.org/x/crypto | v0.50.0 | bcrypt, AES | Phase 2+ (passwords, encryption) [VERIFIED: go module proxy] |
| github.com/prometheus/client_golang | v1.23.2 | Prometheus metrics | Phase 9 (hardening) [VERIFIED: go module proxy] |

### Protobuf Tooling (installed on host)
| Tool | Version | Purpose |
|------|---------|---------|
| protoc | 34.1 (libprotoc) | Proto compiler [VERIFIED: protoc --version] |
| protoc-gen-go | v1.36.11 | Go code generation [VERIFIED: protoc-gen-go --version] |
| protoc-gen-go-grpc | 1.6.0 | gRPC service code generation [VERIFIED: protoc-gen-go-grpc --version] |

### Alternatives Considered
None -- stack is locked by CLAUDE.md. No alternatives to evaluate.

## Architecture Patterns

### Recommended Project Structure (per CLAUDE.md)
```
<service>/
├── api/                          # Proto definitions
│   ├── google/api/               # Google API annotations (if needed)
│   ├── models/                   # Proto model messages
│   └── <service>_api/            # Proto service (RPC methods)
├── cmd/app/main.go               # Entry point, graceful shutdown
├── config/config.go              # Config struct + Load() from config.yaml
├── internal/
│   ├── api/<service>_service_api/ # gRPC handlers (one file per method)
│   ├── bootstrap/                # DI factories for all dependencies
│   ├── models/models.go          # Domain models
│   ├── services/<serviceName>/   # Business logic (one file per method)
│   ├── storage/pgstorage/        # PostgreSQL repository (pgx)
│   ├── pb/                       # Generated protobuf code (gitignored or generated)
│   └── middleware/interceptors.go # gRPC interceptors
├── scripts/generate.sh           # Proto code generation script
├── config.yaml                   # Default local config
├── docker-compose.yaml           # Local infra (PG, Redis, Kafka)
├── Makefile                      # Build, generate, run targets
├── go.mod
└── go.sum
```

### Pattern 1: Config Loading
**What:** Typed config struct loaded from config.yaml via gopkg.in/yaml.v3
**When to use:** Every service at startup
**Example:**
```go
// config/config.go
package config

import (
    "os"
    "gopkg.in/yaml.v3"
)

type Config struct {
    Server   ServerConfig   `yaml:"server"`
    Database DatabaseConfig `yaml:"database"`
    Redis    RedisConfig    `yaml:"redis"`
    Kafka    KafkaConfig    `yaml:"kafka"`
    // Service-specific sections added per service
}

type ServerConfig struct {
    Port int `yaml:"port"`
}

type DatabaseConfig struct {
    DSN string `yaml:"dsn"`
}

type RedisConfig struct {
    Addr     string `yaml:"addr"`
    Password string `yaml:"password"`
    DB       int    `yaml:"db"`
}

type KafkaConfig struct {
    Brokers []string `yaml:"brokers"`
    Topic   string   `yaml:"topic"`
}

func Load(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}
```
[ASSUMED -- standard yaml.v3 pattern, widely used in Go projects]

### Pattern 2: Bootstrap DI Factory
**What:** Bootstrap package creates and wires all dependencies
**When to use:** In main.go to initialize the service
**Example:**
```go
// internal/bootstrap/bootstrap.go
package bootstrap

import (
    "context"
    "log/slog"
    
    "github.com/vbncursed/vkr/auth/config"
    "github.com/vbncursed/vkr/auth/internal/storage/pgstorage"
)

func NewPGStorage(ctx context.Context, cfg *config.Config) (*pgstorage.PGStorage, error) {
    storage, err := pgstorage.New(ctx, cfg.Database.DSN)
    if err != nil {
        return nil, err
    }
    slog.Info("PostgreSQL connected", "dsn_host", "localhost") // never log full DSN
    return storage, nil
}
```
[ASSUMED -- follows Clean Architecture DI pattern from CLAUDE.md and ADR-005]

### Pattern 3: gRPC Service Registration with Stubs
**What:** Register full proto-defined services but return `codes.Unimplemented` from all handlers
**When to use:** Phase 1 skeleton -- all handlers exist but don't do real work yet
**Example:**
```go
// internal/api/auth_service_api/register.go
package auth_service_api

import (
    "context"
    
    pb "github.com/vbncursed/vkr/auth/internal/pb/auth_api"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

func (api *AuthServiceAPI) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
    return nil, status.Error(codes.Unimplemented, "not implemented")
}
```
[ASSUMED -- standard gRPC stub pattern]

### Pattern 4: Graceful Shutdown
**What:** Signal handling for clean service shutdown
**When to use:** Every service's main.go
**Example:**
```go
// cmd/app/main.go
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // ... bootstrap dependencies ...
    
    // Signal handling
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    
    go func() {
        <-sigCh
        slog.Info("shutting down...")
        grpcServer.GracefulStop()
        cancel()
    }()
    
    // Start gRPC server
    lis, _ := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
    grpcServer.Serve(lis)
}
```
[ASSUMED -- standard Go signal handling pattern]

### Pattern 5: Proto File Organization
**What:** Separate proto files for models and service RPCs
**When to use:** Each service's api/ directory
**Example structure for auth:**
```
auth/api/
├── models/
│   └── models.proto          # User, TokenPair messages
└── auth_api/
    └── auth_service.proto    # AuthService RPCs
```
[ASSUMED -- follows CLAUDE.md api/ structure]

### Anti-Patterns to Avoid
- **Monorepo go.mod:** Each service MUST be a separate Go module, not a shared root module
- **ORM usage:** Do NOT use GORM or any ORM -- pgx directly with raw SQL
- **Implementing logic in Phase 1:** Handlers return `codes.Unimplemented` -- no business logic
- **Hardcoded config:** All configuration via config.yaml, never hardcoded in code
- **Panicking on missing optional deps:** Bootstrap should log warnings for unavailable Kafka, not panic

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| UUID generation | Custom UUID | github.com/google/uuid | RFC 4122 compliant, crypto/rand based |
| YAML parsing | Custom config parser | gopkg.in/yaml.v3 | Handles all YAML edge cases |
| gRPC server setup | Raw TCP + custom protocol | google.golang.org/grpc | Standard, battle-tested, interceptor support |
| Proto code generation | Manual message serialization | protoc + plugins | Generated code is correct and fast |
| PostgreSQL connection pooling | Custom pool | pgx/v5/pgxpool | Handles connection lifecycle, health checks |
| Redis commands | Raw TCP to Redis | go-redis/v9 | Handles protocol, pooling, reconnection |

**Key insight:** In Phase 1, "don't hand-roll" primarily applies to infrastructure wiring. The services are skeletons, so the main risk is re-inventing config loading, connection pooling, or gRPC server setup.

## Common Pitfalls

### Pitfall 1: Proto Import Paths
**What goes wrong:** Generated Go code has wrong import paths, packages don't resolve
**Why it happens:** Mismatch between `option go_package` in .proto files and actual module path in go.mod
**How to avoid:** Set `option go_package = "github.com/vbncursed/vkr/<service>/internal/pb/<package>";` matching the go.mod module path exactly
**Warning signs:** `go build` fails with unresolved imports after running generate.sh

### Pitfall 2: Separate Go Modules Not Linking
**What goes wrong:** Services can't import from each other (not needed in Phase 1, but module setup matters)
**Why it happens:** Each service is its own module -- no shared root go.mod
**How to avoid:** Each service's go.mod declares `module github.com/vbncursed/vkr/<service>`. Services don't import each other. TwoFA will import MPC proto in later phases via gRPC clients (generated stubs), not direct Go imports.
**Warning signs:** Attempting to import across service boundaries at compile time

### Pitfall 3: Docker Compose Port Conflicts
**What goes wrong:** Multiple docker-compose files bind the same host ports
**Why it happens:** Each service has its own compose file with PostgreSQL on 5432
**How to avoid:** Use different host port mappings per service (e.g., auth PG: 5433, twofa PG: 5434, mpc PG: 5435). Container port stays 5432.
**Warning signs:** "port already in use" when running multiple services

### Pitfall 4: pgxpool vs pgx Connection
**What goes wrong:** Using single pgx.Conn instead of pgxpool.Pool, blocking under concurrent gRPC calls
**Why it happens:** pgx.Connect creates a single connection, not a pool
**How to avoid:** Use `pgxpool.New(ctx, dsn)` for all services from the start
**Warning signs:** Connection errors under any concurrency

### Pitfall 5: Proto generate.sh Not Setting Output Correctly
**What goes wrong:** Generated .pb.go files end up in wrong directory or with wrong package names
**Why it happens:** protoc --go_out and --go-grpc_out flags need careful path setup
**How to avoid:** Use `--go_out=./internal/pb --go-grpc_out=./internal/pb` with `--go_opt=paths=source_relative --go-grpc_opt=paths=source_relative` and correct `option go_package` in .proto
**Warning signs:** Files generated in unexpected locations, package name mismatches

### Pitfall 6: Kafka Writer Blocking Service Startup
**What goes wrong:** Service hangs during bootstrap trying to connect to unavailable Kafka broker
**Why it happens:** segmentio/kafka-go Writer can block on initial connection
**How to avoid:** Initialize Kafka writer lazily or with a short timeout. Per D-10, log a warning if Kafka is unavailable but don't block startup.
**Warning signs:** Service hangs on startup when Kafka container is not running

## Code Examples

### generate.sh for Auth Service
```bash
#!/bin/bash
# Source: standard protoc invocation pattern [ASSUMED]

set -e

PROTO_DIR="./api"
OUT_DIR="./internal/pb"

mkdir -p "$OUT_DIR"

# Generate Go code from proto files
protoc \
    --proto_path="$PROTO_DIR" \
    --go_out="$OUT_DIR" \
    --go_opt=paths=source_relative \
    --go-grpc_out="$OUT_DIR" \
    --go-grpc_opt=paths=source_relative \
    $(find "$PROTO_DIR" -name "*.proto")

echo "Proto generation complete"
```

### docker-compose.yaml for Auth Service
```yaml
# Source: standard Docker Compose pattern [ASSUMED]
services:
  auth-postgres:
    image: postgres:17
    environment:
      POSTGRES_DB: auth_db
      POSTGRES_USER: auth_user
      POSTGRES_PASSWORD: auth_pass
    ports:
      - "5433:5432"
    volumes:
      - auth-pg-data:/var/lib/postgresql/data

  auth-redis:
    image: redis:8
    ports:
      - "6380:6379"
    command: redis-server --appendonly yes

  auth-kafka:
    image: bitnami/kafka:4.1
    environment:
      KAFKA_CFG_NODE_ID: 1
      KAFKA_CFG_PROCESS_ROLES: broker,controller
      KAFKA_CFG_CONTROLLER_QUORUM_VOTERS: 1@auth-kafka:9093
      KAFKA_CFG_LISTENERS: PLAINTEXT://:9092,CONTROLLER://:9093
      KAFKA_CFG_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092
      KAFKA_CFG_CONTROLLER_LISTENER_NAMES: CONTROLLER
      KAFKA_CFG_LISTENER_SECURITY_PROTOCOL_MAP: CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
    ports:
      - "9092:9092"

volumes:
  auth-pg-data:
```

### PGStorage with initTables (skeleton)
```go
// internal/storage/pgstorage/pgstorage.go [ASSUMED]
package pgstorage

import (
    "context"
    "github.com/jackc/pgx/v5/pgxpool"
)

type PGStorage struct {
    pool *pgxpool.Pool
}

func New(ctx context.Context, dsn string) (*PGStorage, error) {
    pool, err := pgxpool.New(ctx, dsn)
    if err != nil {
        return nil, err
    }
    if err := pool.Ping(ctx); err != nil {
        pool.Close()
        return nil, err
    }
    storage := &PGStorage{pool: pool}
    if err := storage.initTables(ctx); err != nil {
        pool.Close()
        return nil, err
    }
    return storage, nil
}

func (ps *PGStorage) initTables(ctx context.Context) error {
    // Minimal skeleton tables -- expanded in later phases
    _, err := ps.pool.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS users (
            id UUID PRIMARY KEY,
            email VARCHAR(255) UNIQUE NOT NULL,
            password_hash VARCHAR(255) NOT NULL,
            created_at TIMESTAMP NOT NULL DEFAULT NOW(),
            updated_at TIMESTAMP NOT NULL DEFAULT NOW()
        );
    `)
    return err
}

func (ps *PGStorage) Close() {
    ps.pool.Close()
}
```

### Proto Service Definition Example (Auth)
```protobuf
// api/auth_api/auth_service.proto [ASSUMED]
syntax = "proto3";

package auth_api;

option go_package = "auth_api";

import "models/models.proto";

service AuthService {
    rpc Register(RegisterRequest) returns (RegisterResponse);
    rpc Login(LoginRequest) returns (LoginResponse);
    rpc RefreshToken(RefreshTokenRequest) returns (RefreshTokenResponse);
    rpc Logout(LogoutRequest) returns (LogoutResponse);
    rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);
}

message RegisterRequest {
    string email = 1;
    string password = 2;
}

message RegisterResponse {
    string access_token = 1;
    string refresh_token = 2;
}

// ... other request/response messages
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| protoc-gen-go from golang/protobuf | protoc-gen-go from google.golang.org/protobuf | 2020 (APIv2) | Must use `google.golang.org/protobuf` module, not `github.com/golang/protobuf` [VERIFIED: installed plugin is v1.36.11 from google.golang.org/protobuf] |
| pgx/v4 | pgx/v5 | 2022 | pgxpool API changes, `pgxpool.New()` replaces `pgxpool.Connect()` [VERIFIED: v5.9.1 latest] |
| go-redis/v8 | go-redis/v9 | 2023 | Client API changes, context-first methods [VERIFIED: v9.18.0 latest] |
| Zookeeper-based Kafka | KRaft-mode Kafka | Kafka 3.3+ | No Zookeeper needed, use bitnami/kafka with KRaft config [ASSUMED] |
| log package | log/slog | Go 1.21+ | Structured logging is now stdlib, use slog not third-party loggers [VERIFIED: Go 1.26.2 has slog] |

**Deprecated/outdated:**
- `github.com/golang/protobuf`: replaced by `google.golang.org/protobuf` -- do NOT use the old module
- `pgxpool.Connect()`: use `pgxpool.New()` in pgx/v5
- Zookeeper for Kafka: use KRaft mode with `KAFKA_CFG_PROCESS_ROLES`

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | KRaft mode Kafka config with bitnami/kafka:4.1 uses KAFKA_CFG_PROCESS_ROLES env vars | Code Examples | Docker compose won't start Kafka -- easy to fix by checking bitnami docs |
| A2 | `paths=source_relative` with protoc generates files relative to proto source location | Common Pitfalls | Proto generation puts files in wrong directory -- fixable by adjusting proto options |
| A3 | segmentio/kafka-go Writer can block on initial connection to unavailable broker | Common Pitfalls | May not block but could error -- either way D-10 requires graceful handling |

## Open Questions (RESOLVED)

1. **RSA Key Generation for Auth Service**
   - What we know: D-07 says keys go in `auth/keys/`, generated via Makefile target, not committed
   - What's unclear: Exact key size (2048 or 4096 bit) -- CLAUDE.md says RSA-2048 + SHA-256
   - Recommendation: Use 2048-bit as stated in CLAUDE.md, add `make generate-keys` target using `openssl genrsa`
   - RESOLVED: Use RSA-2048 per CLAUDE.md. Auth Makefile `generate-keys` target runs `openssl genrsa 2048`, keys stored in `auth/keys/`, gitignored.

2. **Shared Kafka Instance Across Services**
   - What we know: D-04 allows shared Kafka for local dev
   - What's unclear: Whether to put Kafka in one service's compose or a separate shared compose
   - Recommendation: Include Kafka in auth's docker-compose (first service started), other services reference same broker address. Document that only one needs to be up.
   - RESOLVED: Each service includes its own Kafka in docker-compose with unique host ports (9092/9093/9094). For local dev, only one Kafka instance is needed; services can share a single broker address.

3. **Proto Models Sharing**
   - What we know: D-02 requires proto models match workspace docs
   - What's unclear: Whether services need to share any proto model definitions or each defines its own
   - Recommendation: Each service defines its own models independently. No cross-service proto imports in Phase 1. TwoFA will generate MPC client stubs from MPC's proto in Phase 7.
   - RESOLVED: Each service defines its own proto models independently. No cross-service proto imports. TwoFA generates MPC gRPC client stubs from MPC proto in Phase 7.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | All services | Yes | 1.26.2 | -- |
| protoc | Proto generation | Yes | 34.1 | -- |
| protoc-gen-go | Proto generation | Yes | v1.36.11 | -- |
| protoc-gen-go-grpc | Proto generation | Yes | 1.6.0 | -- |
| Docker | docker-compose infra | Yes | 29.3.1 | -- |
| Docker Compose | docker-compose infra | Yes | v5.1.1 | -- |
| PostgreSQL (via Docker) | Data storage | Via Docker | 17 (image) | -- |
| Redis (via Docker) | Sessions, rate limiting | Via Docker | 8 (image) | -- |
| Kafka (via Docker) | Audit events | Via Docker | 4.1 (image) | -- |

**Missing dependencies with no fallback:** None -- all required tools are available.

**Missing dependencies with fallback:** None.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) + `go test` |
| Config file | None needed -- Go testing is built-in |
| Quick run command | `go test ./... -count=1 -short` (per service directory) |
| Full suite command | `cd <service> && go test ./... -count=1 -v` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| INFRA-01 | Clean Architecture layers compile and wire | smoke | `cd auth && go build ./cmd/app/` | No -- Wave 0 |
| INFRA-02 | Bootstrap factories create dependencies | smoke | `cd auth && go build ./cmd/app/` | No -- Wave 0 |
| INFRA-08 | Config loads from config.yaml | unit | `cd auth && go test ./config/ -run TestLoad` | No -- Wave 0 |
| INFRA-09 | generate.sh produces valid Go code | smoke | `cd auth && bash scripts/generate.sh && go build ./internal/pb/...` | No -- Wave 0 |
| INFRA-10 | Each service is separate Go module | smoke | `cd auth && go mod verify && cd ../twofa && go mod verify && cd ../mpc && go mod verify` | No -- Wave 0 |
| INFRA-11 | Docker Compose starts infrastructure | manual-only | `cd auth && docker compose up -d && docker compose ps` | No -- manual |

### Sampling Rate
- **Per task commit:** `cd <service> && go build ./cmd/app/` (compilation check)
- **Per wave merge:** `cd <service> && go test ./... -count=1 -v` per service
- **Phase gate:** All 3 services compile, generate.sh works, docker-compose up succeeds

### Wave 0 Gaps
- [ ] `auth/config/config_test.go` -- covers INFRA-08 (config loading)
- [ ] `twofa/config/config_test.go` -- covers INFRA-08
- [ ] `mpc/config/config_test.go` -- covers INFRA-08
- [ ] Compilation smoke test is implicit via `go build` -- no separate test file needed

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No (Phase 1 is skeleton) | Stubs only -- implemented in Phase 2-3 |
| V3 Session Management | No (Phase 1 is skeleton) | Stubs only -- implemented in Phase 3 |
| V4 Access Control | No (Phase 1 is skeleton) | Stubs only |
| V5 Input Validation | No (Phase 1 is skeleton) | No real inputs processed |
| V6 Cryptography | No (Phase 1 is skeleton) | Config references keys but no crypto operations |

### Known Threat Patterns for Phase 1

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Secrets in config.yaml committed to git | Information Disclosure | Add `keys/` and sensitive configs to .gitignore |
| Docker Compose default passwords in repo | Information Disclosure | Acceptable for local dev only, document as non-production |
| Proto definitions exposing internal structure | Information Disclosure | Not a concern -- proto is the API contract, not implementation |

**Phase 1 security scope is minimal:** No real authentication, no real data processing. Primary concern is ensuring sensitive paths (RSA keys, encryption keys) are gitignored from the start.

## Sources

### Primary (HIGH confidence)
- Go module proxy -- verified all dependency versions via `go list -m @latest`
- Local toolchain -- verified Go 1.26.2, protoc 34.1, plugins, Docker via CLI
- CLAUDE.md -- project structure, conventions, constraints
- workspace/02 - Services/*.md -- service API definitions for proto contracts
- workspace/04 - Decisions/ADR Log.md -- architectural decisions

### Secondary (MEDIUM confidence)
- None needed -- this phase uses well-established Go patterns

### Tertiary (LOW confidence)
- bitnami/kafka Docker image KRaft configuration (A1) -- verify against bitnami docs if issues arise

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all versions verified against Go module proxy
- Architecture: HIGH -- structure is fully specified in CLAUDE.md and ADR-005
- Pitfalls: HIGH -- common Go/gRPC/Docker patterns, well-documented issues

**Research date:** 2026-04-11
**Valid until:** 2026-05-11 (30 days -- stable domain, no fast-moving dependencies)
