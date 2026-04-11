# Technology Stack

**Analysis Date:** 2026-04-11

## Languages

**Primary:**
- Go 1.26.2 - All backend microservices (Gateway, Auth, TwoFA, MPC nodes)

## Runtime

**Environment:**
- Go 1.26.2 runtime

**Package Manager:**
- Go modules (`go.mod`, `go.sum`)
- Lockfile: Required for reproducible builds

## Frameworks

**Core:**
- gRPC - Inter-service communication (Gateway ↔ Auth, TwoFA; TwoFA ↔ MPC nodes)
- protobuf - RPC method definitions and data models (`api/` directories in each service)

**Protocol Translation:**
- Custom REST-to-gRPC translation layer in `gateway/internal/api/` - HTTP endpoints translate to gRPC calls

**Build/Dev:**
- Makefile - Build automation
- Bash scripts - Protobuf code generation (`scripts/generate.sh` in each service)

## Key Dependencies

**Critical:**
- `google.golang.org/grpc` [version TBD] - gRPC framework and libraries
- `google.golang.org/protobuf` [version TBD] - Protocol buffers runtime
- `github.com/jackc/pgx/v5` - PostgreSQL driver (direct, no ORM)
- `github.com/redis/go-redis/v9` - Redis client
- `github.com/segmentio/kafka-go` - Kafka producer/consumer
- `github.com/golang-jwt/jwt/v5` - JWT token generation and validation (RS256)
- `golang.org/x/crypto` - Cryptographic functions (bcrypt password hashing, TOTP, AES-256-GCM)

**Infrastructure:**
- `github.com/prometheus/client_golang` - Metrics collection
- `github.com/google/uuid` - UUID generation for entities
- `gopkg.in/yaml.v3` - YAML config file parsing

**Testing:**
- Standard `testing` package (Go stdlib)
- No external testing framework specified

## Configuration

**Environment:**
- YAML-based configuration files (`config.yaml` in each service)
- Each service loads from `config/config.go`
- Configuration includes: database connection, Redis, Kafka brokers, JWT keys, server ports, environment (dev/prod)
- Supports environment variable overrides (standard Go practice)

**Build:**
- Protobuf generation: `scripts/generate.sh`
- Generated code location: `internal/pb/` (generated from `api/` proto definitions)
- Google API annotations for HTTP mappings: `api/google/api/` (annotations.proto, field_behavior.proto, http.proto)

## Platform Requirements

**Development:**
- Go 1.26.2 installed
- Protoc compiler (for generating code from .proto files)
- Docker (for running PostgreSQL, Redis, Kafka, Prometheus, Grafana locally)
- Make (for Makefile commands)

**Production:**
- Deployment target: Containerized (Docker images for each service)
- Kubernetes-ready (gRPC Health Check Protocol implemented in each service)
- Linux/Darwin/Windows (Go cross-platform)

## Service-Specific Stack

**API Gateway (`gateway/`):**
- HTTP server for REST endpoints
- gRPC client connections to Auth and TwoFA services
- Redis client for rate limiting counters
- Middleware: JWT validation, rate limiting, CORS, request logging

**Auth Service (`auth/`):**
- gRPC server
- PostgreSQL for user accounts and session audit logs
- Redis for refresh token storage (TTL 7 days)
- Kafka producer for user registration/login events
- JWT RS256 token generation

**TwoFA Service (`twofa/`):**
- gRPC server
- PostgreSQL for 2FA metadata and audit
- gRPC client connections to 3× MPC nodes
- Kafka producer for 2FA events
- Shamir Secret Sharing implementation (2-of-3, custom GF(256))
- TOTP verification (RFC 6238)

**MPC Nodes (`mpc/`, 3 instances):**
- gRPC server
- PostgreSQL for share storage (encrypted at-rest with AES-256-GCM)
- Kafka producer for audit events
- No outgoing service dependencies

---

*Stack analysis: 2026-04-11*
