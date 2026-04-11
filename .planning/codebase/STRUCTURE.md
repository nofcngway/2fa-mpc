# Codebase Structure

**Analysis Date:** 2026-04-11

## Directory Layout

```
2fa/
├── gateway/                    # API Gateway service (HTTP entry point)
├── auth/                       # Authentication service (user management, JWT)
├── twofa/                      # 2FA orchestration (Shamir split/combine, TOTP)
├── mpc/                        # MPC node (share storage, encryption)
├── migration/                  # Cross-service database migrations
├── monitoring/                 # Prometheus + Grafana configuration
├── workspace/                  # Obsidian vault (architecture decisions, notes)
├── .planning/codebase/         # GSD analysis documents
├── CLAUDE.md                   # Project specifications and rules
├── TZ.md                       # Technical requirements (detailed specs)
└── .obsidian/                  # Obsidian vault config
```

## Directory Purposes

**gateway/:**
- Purpose: Single HTTP REST entry point for frontend clients, translates REST → gRPC, enforces rate limiting
- Contains: gRPC service API handlers, HTTP route definitions, middleware
- Key files: `cmd/app/main.go`, `internal/api/`, `config.yaml`

**auth/:**
- Purpose: User authentication, JWT token lifecycle management, session handling
- Contains: Registration, login, token refresh/validation, password validation logic
- Key files: `cmd/app/main.go`, `internal/services/authService/`, `internal/storage/pgstorage/`, `internal/storage/redisstorage/`

**twofa/:**
- Purpose: 2FA orchestration, Shamir Secret Sharing implementation, TOTP validation, MPC node coordination
- Contains: 2FA setup/verify/disable operations, custom Shamir and TOTP implementations
- Key files: `cmd/app/main.go`, `internal/services/twofaService/`, `internal/services/twofaService/shamir/`, `internal/services/twofaService/totp/`

**mpc/:**
- Purpose: Distributed secret share storage with encryption at rest, one instance per share (3 total)
- Contains: Share encryption/decryption, PostgreSQL persistence, access control
- Key files: `cmd/app/main.go`, `internal/services/mpcService/`, `internal/storage/pgstorage/`

**migration/:**
- Purpose: SQL schema migrations shared across services
- Contains: SQL files for creating tables (users, sessions, 2fa_records, shares, audit_log)
- Key files: `*.sql` migration files

**monitoring/:**
- Purpose: Prometheus scrape configuration and Grafana dashboards
- Contains: prometheus.yml, dashboard JSON definitions
- Key files: `prometheus.yml`, `dashboards/`

**workspace/:**
- Purpose: Obsidian vault for architecture documentation, decisions, and project progress
- Contains: Architecture notes, service documentation, security protocols, decision logs
- Structure:
  - `00 - Index.md`: Navigation
  - `01 - Architecture/`: System overview, service interactions, data flows
  - `02 - Services/`: Individual service documentation
  - `03 - Security/`: Protocol specifications (Shamir, TOTP, AES, JWT)
  - `04 - Decisions/`: ADR (Architecture Decision Records)
  - `05 - Progress/`: TODO list, changelog

**.planning/codebase/:**
- Purpose: GSD (Generalized System Design) analysis documents
- Contains: ARCHITECTURE.md, STRUCTURE.md, CONVENTIONS.md, TESTING.md, CONCERNS.md, INTEGRATIONS.md, STACK.md

## Key File Locations

**Entry Points:**
- `gateway/cmd/app/main.go`: HTTP REST gateway server startup
- `auth/cmd/app/main.go`: Authentication service startup
- `twofa/cmd/app/main.go`: 2FA service startup
- `mpc/cmd/app/main.go`: MPC node service startup

**Configuration:**
- `gateway/config.yaml`: Gateway configuration (HTTP port, gRPC targets, rate limits)
- `auth/config.yaml`: Auth service configuration (gRPC port, DB, Redis, Kafka, JWT keys)
- `twofa/config.yaml`: TwoFA service configuration (gRPC port, DB, MPC node addresses, rate limits)
- `mpc/config.yaml`: MPC node configuration (gRPC port, encryption key, node ID)

**Protocol Definitions:**
- `gateway/api/google/api/`: Google API annotations (http.proto, field_behavior.proto)
- `gateway/api/models/`: Shared proto models (gateway_model.proto)
- `gateway/api/gateway_api/`: Gateway service RPC definitions (gateway.proto)
- `auth/api/google/api/`: Google API annotations
- `auth/api/models/`: Auth proto models (auth_model.proto, user, session, token)
- `auth/api/auth_api/`: Auth service RPC definitions (auth.proto)
- `twofa/api/google/api/`: Google API annotations
- `twofa/api/models/`: TwoFA proto models (twofa_model.proto, share, 2fa_record)
- `twofa/api/twofa_api/`: TwoFA service RPC definitions (twofa.proto)
- `mpc/api/google/api/`: Google API annotations
- `mpc/api/models/`: MPC proto models (mpc_model.proto, share)
- `mpc/api/mpc_api/`: MPC service RPC definitions (mpc.proto)

**Core Business Logic:**
- `auth/internal/services/authService/auth_service.go`: Auth service structure and interface
- `auth/internal/services/authService/register.go`: User registration logic
- `auth/internal/services/authService/login.go`: User login and JWT generation
- `auth/internal/services/authService/refresh.go`: Token refresh/rotation
- `auth/internal/services/authService/validate.go`: Token validation
- `auth/internal/services/authService/password_validation.go`: Password policy enforcement
- `auth/internal/services/authService/jwt.go`: JWT generation and parsing (RS256)

- `twofa/internal/services/twofaService/twofa_service.go`: TwoFA service structure and interface
- `twofa/internal/services/twofaService/setup.go`: 2FA setup and secret splitting
- `twofa/internal/services/twofaService/verify.go`: 2FA verification and secret reconstruction
- `twofa/internal/services/twofaService/disable.go`: 2FA disabling and share cleanup
- `twofa/internal/services/twofaService/shamir/shamir.go`: Shamir Secret Sharing (GF(256))
- `twofa/internal/services/twofaService/totp/totp.go`: TOTP generation and validation (RFC 6238)
- `twofa/internal/services/twofaService/rate_limit.go`: Rate limiting (5 attempts per 5 min)

- `mpc/internal/services/mpcService/mpc_service.go`: MPC service structure
- `mpc/internal/services/mpcService/store_share.go`: Share encryption and storage
- `mpc/internal/services/mpcService/retrieve_share.go`: Share retrieval and decryption
- `mpc/internal/services/mpcService/delete_share.go`: Share deletion
- `mpc/internal/services/mpcService/encryption.go`: AES-256-GCM encryption/decryption

**gRPC Handlers:**
- `gateway/internal/api/gateway_service_api/gateway_api.go`: Gateway handler wrapper
- `gateway/internal/api/gateway_service_api/register.go`: Register RPC handler
- `gateway/internal/api/gateway_service_api/login.go`: Login RPC handler
- `gateway/internal/api/gateway_service_api/refresh_token.go`: RefreshToken RPC handler
- `gateway/internal/api/gateway_service_api/logout.go`: Logout RPC handler
- `gateway/internal/api/gateway_service_api/setup_2fa.go`: Setup2FA RPC handler
- `gateway/internal/api/gateway_service_api/verify_2fa.go`: Verify2FA RPC handler
- `gateway/internal/api/gateway_service_api/disable_2fa.go`: Disable2FA RPC handler
- `gateway/internal/api/gateway_service_api/get_2fa_status.go`: Get2FAStatus RPC handler

- `auth/internal/api/auth_service_api/auth_api.go`: Auth handler wrapper
- `auth/internal/api/auth_service_api/register.go`: Register RPC handler
- `auth/internal/api/auth_service_api/login.go`: Login RPC handler
- `auth/internal/api/auth_service_api/refresh_token.go`: RefreshToken RPC handler
- `auth/internal/api/auth_service_api/logout.go`: Logout RPC handler
- `auth/internal/api/auth_service_api/validate_token.go`: ValidateToken RPC handler

- `twofa/internal/api/twofa_service_api/twofa_api.go`: TwoFA handler wrapper
- `twofa/internal/api/twofa_service_api/setup_2fa.go`: Setup2FA RPC handler
- `twofa/internal/api/twofa_service_api/verify_2fa.go`: Verify2FA RPC handler
- `twofa/internal/api/twofa_service_api/disable_2fa.go`: Disable2FA RPC handler
- `twofa/internal/api/twofa_service_api/get_2fa_status.go`: Get2FAStatus RPC handler

- `mpc/internal/api/mpc_service_api/mpc_api.go`: MPC handler wrapper
- `mpc/internal/api/mpc_service_api/store_share.go`: StoreShare RPC handler
- `mpc/internal/api/mpc_service_api/retrieve_share.go`: RetrieveShare RPC handler
- `mpc/internal/api/mpc_service_api/delete_share.go`: DeleteShare RPC handler

**Data Access (Storage):**
- `<service>/internal/storage/pgstorage/pgstorage.go`: PostgreSQL connection pool initialization, table creation
- `<service>/internal/storage/pgstorage/models.go`: Storage-layer data models and SQL constants
- `auth/internal/storage/pgstorage/user.go`: User CRUD operations (Create, GetByEmail, GetByID)
- `auth/internal/storage/pgstorage/session.go`: Session/audit log operations
- `twofa/internal/storage/pgstorage/twofa_record.go`: 2FA metadata operations
- `mpc/internal/storage/pgstorage/share.go`: Encrypted share CRUD operations

- `auth/internal/storage/redisstorage/redisstorage.go`: Redis client initialization
- `auth/internal/storage/redisstorage/session.go`: Refresh token operations (Set, Get, Delete with TTL)

**Dependency Injection:**
- `<service>/internal/bootstrap/bootstrap.go`: Main bootstrap coordinator (creates all components)
- `<service>/internal/bootstrap/auth_service.go`: AuthService factory
- `<service>/internal/bootstrap/twofa_service.go`: TwoFAService factory
- `<service>/internal/bootstrap/mpc_service.go`: MPCService factory
- `<service>/internal/bootstrap/pgstorage.go`: PostgreSQL storage factory
- `<service>/internal/bootstrap/redisstorage.go`: Redis storage factory
- `<service>/internal/bootstrap/kafka_producer.go`: Kafka producer factory
- `<service>/internal/bootstrap/auth_api.go`: AuthServiceAPI factory
- `<service>/internal/bootstrap/twofa_api.go`: TwoFAServiceAPI factory
- `<service>/internal/bootstrap/mpc_api.go`: MPCServiceAPI factory
- `<service>/internal/bootstrap/server.go`: gRPC server initialization with interceptors

**Middleware & Interceptors:**
- `<service>/internal/middleware/interceptors.go`: gRPC unary/stream interceptors (logging, metrics, recovery, auth)

**Kafka Consumers (Event Processing):**
- `<service>/internal/consumer/`: Directory for event consumers (if needed for cross-service events)

**Tests:**
- `<service>/internal/services/<serviceName>/<operation>_test.go`: Unit tests for business logic
- `<service>/internal/services/authService/password_validation_test.go`: Password validation tests
- `<service>/internal/api/<service>_service_api/<operation>_test.go`: Handler integration tests
- `<service>/internal/storage/pgstorage/<entity>_test.go`: Storage layer tests

**Build & Deployment:**
- `<service>/Makefile`: Build targets (generate proto, build, test, docker)
- `<service>/scripts/generate.sh`: Protocol buffer code generation script
- `<service>/scripts/command.mk`: Common make commands
- `<service>/docker-compose.yaml`: Local development environment (PostgreSQL, Redis)
- `<service>/go.mod`: Go module definition
- `<service>/go.sum`: Go dependency lock file

**Generated Code:**
- `<service>/internal/pb/`: Generated protobuf code (never edit manually)
  - `models/`: Generated message code
  - `auth_api/`, `twofa_api/`, `mpc_api/`, `gateway_api/`: Generated service stubs

**Configuration & Models:**
- `<service>/config/config.go`: Configuration loading from YAML
- `<service>/internal/models/models.go`: Domain model structs (User, Session, TokenPair, etc.)

## Naming Conventions

**Files:**
- `*.proto`: Protocol buffer definitions (snake_case, e.g., `auth_model.proto`, `auth.proto`)
- `*.go`: Go source files (snake_case, e.g., `password_validation.go`, `store_share.go`)
- `*_test.go`: Unit/integration tests (same name as file being tested with `_test` suffix)
- `*_pb.go`: Generated protobuf code (auto-generated, never edited)
- `*_grpc.pb.go`: Generated gRPC stubs (auto-generated, never edited)
- `config.yaml`: Service configuration (always lowercase)
- `docker-compose.yaml`: Docker Compose configuration
- `Makefile`: Build automation
- `go.mod`, `go.sum`: Go module files

**Directories:**
- `api/`: Protocol buffer definitions (grouped by service)
- `cmd/`: Command-line entry points (always `cmd/app/main.go`)
- `config/`: Configuration loading (always `config/config.go`)
- `internal/`: Private package tree (never imported by external packages)
  - `api/`: gRPC service handlers
  - `bootstrap/`: Dependency injection factories
  - `models/`: Domain models
  - `services/`: Business logic
  - `storage/`: Data access (repositories)
  - `middleware/`: gRPC interceptors
  - `consumer/`: Event consumers
  - `pb/`: Generated protobuf code
- `scripts/`: Helper scripts (always include `generate.sh`)

**Functions:**
- camelCase (e.g., `registerUser`, `validatePassword`, `storeShare`)
- Service methods: action verbs (Register, Login, Setup2FA, Verify2FA)
- Repository methods: CRUD pattern (Create, Get, Update, Delete)
- Utility functions: descriptive (ValidateEmail, GenerateJWT, EncryptShare)

**Variables:**
- camelCase (e.g., `userID`, `passwordHash`, `encryptionKey`)
- Constants: UPPER_SNAKE_CASE (e.g., `MAX_PASSWORD_LENGTH`, `JWT_ACCESS_EXPIRY`)
- Interface names: PascalCase with suffix -er (e.g., `UserRepository`, `TokenValidator`)
- Struct names: PascalCase (e.g., `User`, `Session`, `TokenPair`)

**Types:**
- Structs: PascalCase (e.g., `User`, `Session`, `RegisterRequest`, `LoginResponse`)
- Interfaces: PascalCase with -er suffix (e.g., `UserRepository`, `PasswordValidator`)
- Error types: PascalCase with Error suffix (e.g., `InvalidPasswordError`)

**Proto messages:**
- PascalCase (e.g., `User`, `RegisterRequest`, `RegisterResponse`, `TokenPair`)
- Services: PascalCase with Service suffix (e.g., `AuthService`, `TwoFAService`, `MPCNodeService`)
- RPC methods: PascalCase (e.g., `Register`, `Login`, `Setup2FA`, `Verify2FA`)

## Where to Add New Code

**New Feature (e.g., new Auth operation):**
- Primary code: `<service>/internal/services/<serviceName>/<operation>.go` (business logic)
- Handler: `<service>/internal/api/<service>_service_api/<operation>.go` (gRPC handler)
- Proto: `<service>/api/<service>_api/<service>.proto` (add RPC method and messages)
- Tests: `<service>/internal/services/<serviceName>/<operation>_test.go` (unit tests)
- Storage changes: `<service>/internal/storage/pgstorage/<entity>.go` (if new persistence needed)

**New Service (e.g., notification service):**
- Create directory at project root: `notification/`
- Follow Clean Architecture structure:
  ```
  notification/
  ├── api/
  │   ├── google/api/
  │   ├── models/
  │   │   └── notification_model.proto
  │   └── notification_api/
  │       └── notification.proto
  ├── cmd/app/main.go
  ├── config/config.go
  ├── internal/
  │   ├── api/notification_service_api/
  │   ├── bootstrap/
  │   ├── models/models.go
  │   ├── services/notificationService/
  │   ├── storage/pgstorage/
  │   ├── middleware/interceptors.go
  │   └── pb/
  ├── scripts/generate.sh
  ├── config.yaml
  ├── docker-compose.yaml
  ├── Makefile
  ├── go.mod
  └── go.sum
  ```
- Register new service in Gateway routing
- Update workspace documentation (`workspace/02 - Services/<NewService>.md`)

**New Component/Module (e.g., backup code manager):**
- Location: `<service>/internal/services/<serviceName>/backup_codes/` (if service-specific)
- Or: `shared/backup_codes/` (if used across multiple services)
- Pattern: Group related functionality in subdirectory under services

**Utilities & Helpers:**
- Shared helpers: `shared/utils/` (e.g., `shared/utils/crypto/`, `shared/utils/validation/`)
- Service-specific: `<service>/internal/services/<serviceName>/<utility>.go` (inline for simple utilities)

**Tests:**
- Unit tests: `<service>/internal/<layer>/<module>/<operation>_test.go` (same directory as code)
- Integration tests: `<service>/tests/integration/` (optional, for complex scenarios)
- Test fixtures: `<service>/tests/fixtures/` (test data, factories, mocks)

## Special Directories

**generated code (internal/pb/):**
- Purpose: Holds all protobuf-generated code
- Generated by: `scripts/generate.sh` (buf or protoc)
- Committed: Yes (for reproducible builds)
- How to regenerate: Run `make generate` or `./scripts/generate.sh`
- Never edit manually: All files in this directory are auto-generated

**workspace/ (Obsidian vault):**
- Purpose: Architecture decisions, service documentation, progress tracking
- Generated: No (manually written)
- Committed: Yes (part of project documentation)
- Structure: 5 main sections (Architecture, Services, Security, Decisions, Progress)

**.obsidian/ (Vault config):**
- Purpose: Obsidian editor configuration
- Generated: Partially (plugins, themes)
- Committed: Yes (maintains vault setup)

**migration/:**
- Purpose: SQL schema migrations
- Contains: Numbered SQL files (001_initial_schema.sql, 002_add_2fa_tables.sql)
- Committed: Yes (must be in version control)
- How to run: Migration tool in each service startup (see `internal/storage/pgstorage/pgstorage.go`)

**monitoring/:**
- Purpose: Prometheus and Grafana configuration
- Contains: prometheus.yml (scrape configs), dashboards/ (JSON definitions)
- Committed: Yes
- How to update: Modify YAML/JSON, restart Docker Compose services

**docker-compose.yaml (service-level):**
- Purpose: Local development environment for single service
- Contains: PostgreSQL, Redis, optional Kafka connector
- Committed: Yes
- Usage: `docker-compose up` during development

---

*Structure analysis: 2026-04-11*
