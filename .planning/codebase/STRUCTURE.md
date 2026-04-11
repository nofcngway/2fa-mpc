# Codebase Structure

**Analysis Date:** 2026-04-11

## Directory Layout

```
/Users/vbncursed/programming/2fa/
├── auth/                     # Auth Service (registration, login, JWT)
├── gateway/                  # API Gateway (REST → gRPC translator)
├── twofa/                    # TwoFA Service (2FA orchestration, Shamir)
├── mpc/                      # MPC Node Service (share storage with encryption)
├── migration/                # Cross-service database migrations (if needed)
├── monitoring/               # Prometheus/Grafana configuration
├── workspace/                # Obsidian vault (ADR, decisions, progress, security docs)
├── .planning/
│   └── codebase/            # Generated planning documents (ARCHITECTURE.md, STRUCTURE.md, etc.)
├── CLAUDE.md                # Project rules and conventions (Russian)
├── README.md                # Development prompts and target state
├── TZ.md                    # Technical specification (Russian, copy of requirements)
└── .obsidian/               # Obsidian workspace configuration
```

## Directory Purposes

**auth/ - Authentication Service:**
- Purpose: User registration, login, JWT token management, session validation
- Contains: Complete Go service with Clean Architecture
- Key files:
  - `cmd/app/main.go`: Service entry point
  - `api/auth_api/auth.proto`: gRPC service definition
  - `internal/services/authService/`: Business logic (register, login, JWT, password validation)
  - `internal/storage/pgstorage/`: PostgreSQL repository for users and sessions
  - `internal/storage/redisstorage/`: Redis repository for refresh tokens
  - `config/config.go`: Configuration loader from config.yaml

**gateway/ - API Gateway:**
- Purpose: Single HTTP entry point, REST-to-gRPC translation, rate limiting
- Contains: Go service exposing HTTP endpoints
- Key files:
  - `cmd/app/main.go`: Gateway entry point
  - HTTP handlers: register, login, refresh, logout, 2fa endpoints
  - gRPC clients: connections to Auth and TwoFA services
  - Middleware: JWT validation, rate limiting, CORS, logging

**twofa/ - Two-Factor Authentication Service:**
- Purpose: 2FA setup/verification, Shamir secret sharing orchestration, MPC client coordination
- Contains: Go service with cryptographic operations
- Key files:
  - `cmd/app/main.go`: Service entry point
  - `api/twofa_api/twofa.proto`: TwoFA service definition
  - `api/mpc_api/mpc.proto`: MPC client contract (shared with mpc/)
  - `internal/services/twofaService/shamir/`: Shamir Secret Sharing in GF(256)
  - `internal/services/twofaService/totp/`: TOTP generation and validation (RFC 6238)
  - `internal/clients/mpc/`: gRPC clients to 3 MPC nodes
  - `internal/storage/pgstorage/`: Metadata storage (2FA records, backup codes)

**mpc/ - MPC Node Service:**
- Purpose: Secure storage of secret shares with at-rest encryption
- Contains: Go service (3 identical instances, NODE_ID from config distinguishes them)
- Key files:
  - `cmd/app/main.go`: Node entry point
  - `api/mpc_api/mpc.proto`: MPC service definition (copy from twofa/)
  - `internal/services/shareService/`: Share storage/retrieval logic
  - `internal/crypto/aes.go`: AES-256-GCM encryption/decryption
  - `internal/storage/pgstorage/`: Share persistence
  - Middleware: Authorization via metadata (shared secret token)

**migration/:**
- Purpose: Cross-service database migrations (if needed)
- Contains: SQL migration files (currently empty)
- Usage: For schema changes that span multiple services

**monitoring/:**
- Purpose: Prometheus and Grafana configuration
- Contains: prometheus.yml, grafana dashboards (currently empty)
- Usage: Collect and visualize metrics from all services

**workspace/ - Obsidian Vault:**
- Purpose: Architecture documentation, security protocols, decisions, progress tracking
- Structure:
  - `00 - Index.md`: Navigation hub
  - `01 - Architecture/`: System overview, data flows, service descriptions
  - `02 - Services/`: Per-service documentation (API endpoints, responsibilities)
  - `03 - Security/`: Cryptographic protocols (Shamir, TOTP, AES, JWT, password policy)
  - `04 - Decisions/`: Architecture Decision Records (ADR Log.md)
  - `05 - Progress/`: TODO.md, Changelog.md for sprint tracking

## Key File Locations

**Entry Points:**
- `auth/cmd/app/main.go`: Auth Service binary entry point
- `gateway/cmd/app/main.go`: API Gateway binary entry point
- `twofa/cmd/app/main.go`: TwoFA Service binary entry point
- `mpc/cmd/app/main.go`: MPC Node Service binary entry point

**Configuration:**
- `<service>/config.yaml`: YAML config file per service (database, Redis, Kafka, ports, encryption key)
- `<service>/config/config.go`: Config struct and loader using gopkg.in/yaml.v3
- `<service>/docker-compose.yaml`: Local development infrastructure per service

**Core Logic:**
- `auth/internal/services/authService/`: Registration, login, JWT generation, token refresh
- `twofa/internal/services/twofaService/`: 2FA operations, Shamir split/combine, TOTP validation
- `mpc/internal/services/shareService/`: Share encryption, storage, retrieval
- `gateway/handlers/`: HTTP endpoint handlers that translate to gRPC calls

**Data Access:**
- `<service>/internal/storage/pgstorage/`: PostgreSQL repository (pgx client, CRUD methods)
- `<service>/internal/storage/redisstorage/`: Redis repository (session/rate limit storage)
- `twofa/internal/clients/mpc/`: gRPC clients to MPC nodes with timeout handling

**Testing:**
- `<service>/internal/services/<service>/..._test.go`: Unit tests for business logic
- Examples: `auth/internal/services/authService/password_validation_test.go`, `twofa/internal/services/twofaService/shamir/shamir_test.go`

**Cryptography:**
- `twofa/internal/services/twofaService/shamir/shamir.go`: Shamir Secret Sharing (Split, Combine)
- `twofa/internal/services/twofaService/shamir/gf256.go`: GF(256) arithmetic (Add, Multiply, Inverse)
- `twofa/internal/services/twofaService/totp/totp.go`: TOTP generation, validation, provisioning URI
- `mpc/internal/crypto/aes.go`: AES-256-GCM encrypt/decrypt with nonce

**Protobuf Definitions:**
- `auth/api/auth_api/auth.proto`: Auth service methods (Register, Login, RefreshToken, Logout, ValidateToken)
- `twofa/api/twofa_api/twofa.proto`: TwoFA service methods (Setup2FA, Verify2FA, Disable2FA, Get2FAStatus)
- `twofa/api/mpc_api/mpc.proto`: MPC service contract (StoreShare, RetrieveShare, DeleteShare)
- `mpc/api/mpc_api/mpc.proto`: Copy of twofa contract
- `<service>/api/models/`: Proto message definitions (User, TokenPair, Share, TwoFARecord, etc.)
- `<service>/api/google/api/`: Google API annotations for proto

**Bootstrap & DI:**
- `<service>/internal/bootstrap/`: Dependency injection factories
  - `auth_service.go`: Create AuthService with dependencies
  - `pgstorage.go`: Initialize pgx connection pool
  - `redis.go`: Initialize Redis client
  - `mpc_clients.go`: Create 3 gRPC clients to MPC nodes (TwoFA only)
  - `kafka_producer.go`: Initialize Kafka producer
  - `server.go`: Register gRPC handlers and middleware

**Middleware:**
- `<service>/internal/middleware/interceptors.go`: gRPC interceptors
  - Logging interceptor (request/response, slog structured)
  - Metrics interceptor (Prometheus counters and histograms)
  - Recovery interceptor (panic handling)
  - Authorization interceptor (MPC nodes only)

**Generated Protobuf Code:**
- `<service>/internal/pb/`: Generated .pb.go and _grpc.pb.go files (auto-generated by protoc)

## Naming Conventions

**Files:**
- Service logic: `<operation>.go` (e.g., `register.go`, `login.go`, `setup.go`, `verify.go`)
- Tests: `<operation>_test.go` (e.g., `password_validation_test.go`, `shamir_test.go`)
- Models: `models.go` (domain models per service)
- Repositories: `<entity>.go` (e.g., `user.go`, `session.go`, `share.go`)
- Clients: `client.go` (gRPC clients to other services)
- Configuration: `config.go`, `config.yaml`
- Middleware: `interceptors.go`

**Directories:**
- Service packages: camelCase, single word (e.g., `authService`, `twofaService`, `shareService`)
- Layer directories: snake_case (e.g., `auth_service_api`, `pgstorage`, `redisstorage`)
- Utilities: plural form (e.g., `services/`, `storage/`, `clients/`)

**Functions & Interfaces:**
- Public (exported): PascalCase (e.g., `Register`, `CreateUser`, `StoreShare`)
- Private (unexported): camelCase (e.g., `validateEmail`, `zeroizeSecret`)
- Interfaces: descriptive, usually ending in "er" or explicit (e.g., `Storage`, `Producer`, `Encryptor`, `MPCClient`)

**Variables:**
- Constants: UPPER_SNAKE_CASE (e.g., `DEFAULT_TTL`, `GCM_NONCE_SIZE`)
- Package-level: camelCase or PascalCase for exported
- Method receivers: short (e.g., `s *AuthService`, `r *Repository`)

## Where to Add New Code

**New Authentication Feature:**
- Primary code: `auth/internal/services/authService/<feature>.go`
- Database schema: Update `auth/internal/storage/pgstorage/models.go` and `initTables()` in `pgstorage.go`
- Tests: `auth/internal/services/authService/<feature>_test.go`
- gRPC method: Add to `auth/api/auth_api/auth.proto` → implement handler in `auth/internal/api/auth_service_api/<feature>.go`
- Kafka event: Call `kafkaProducer.Publish()` in service logic

**New 2FA Feature:**
- Primary code: `twofa/internal/services/twofaService/<feature>.go`
- MPC orchestration: Call `mpcClients[].StoreShare()` or `RetrieveShare()` from service logic
- Cryptography: Add to `twofa/internal/services/twofaService/shamir/` or `totp/`
- Tests: Shamir tests in `shamir_test.go`, TOTP tests in `totp_test.go`

**New MPC Operation:**
- Primary code: `mpc/internal/services/shareService/<operation>.go`
- Encryption: Use `encryptor.Encrypt()` and `Decrypt()` from `mpc/internal/crypto/aes.go`
- Storage: Call `storage.CreateShare()` or `GetShare()` from `mpc/internal/storage/pgstorage/share.go`

**New HTTP Endpoint (Gateway only):**
- HTTP handler: `gateway/handlers/<feature>.go`
- gRPC client call: Use `authClient` or `twofaClient` from bootstrap
- Middleware: Register rate limiter in endpoint middleware chain
- Response: Translate gRPC response to JSON, map status codes to HTTP

**Shared Utilities:**
- Validation helpers: `<service>/internal/services/<serviceName>/<helper>.go` (can be reused across handlers/services)
- Constants: Define in `internal/models/models.go` or respective file
- Custom types: Define in `internal/models/models.go`

## Special Directories

**workspace/ - Obsidian Vault:**
- Purpose: Design documentation and progress tracking
- Generated: No (manually created)
- Committed: Yes (part of git repo)
- Files are Markdown with Obsidian wikilinks for cross-references
- Update when: Adding new services, making architectural decisions, security protocol changes

**.planning/codebase/:**
- Purpose: Generated analysis documents for GSD (code generation system)
- Generated: Yes (by GSD mapping agents)
- Committed: Yes
- Contains: ARCHITECTURE.md, STRUCTURE.md, CONVENTIONS.md, TESTING.md, CONCERNS.md

**internal/pb/:**
- Purpose: Auto-generated protobuf code
- Generated: Yes (by `protoc` via `scripts/generate.sh`)
- Committed: Yes (generated code is committed, NOT in .gitignore)
- Do not edit: Regenerate via `make proto` or `scripts/generate.sh`

**internal/api/<service>_service_api/:**
- Purpose: gRPC handlers (translation layer between proto and business logic)
- Generated: No (manually implemented)
- Contains: One file per RPC method, each handler calls service layer

---

*Structure analysis: 2026-04-11*
