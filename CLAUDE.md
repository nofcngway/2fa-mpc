# MPC-2FA — Двухфакторная аутентификация с распределенным хранением секретов

## Проект

: Разработка двухфакторной системы аутентификации с распределенным хранением секретов и использованием протоколов безопасных многосторонних вычислений.

## Стек

- **Go** 1.26.2
- **PostgreSQL** — users, sessions, audit, 2FA metadata, shares
- **Redis** 8.6.2 — refresh-сессии, rate limiting
- **Kafka** 4.1.2 — события аудита между сервисами
- **Prometheus** + **Grafana** — мониторинг и метрики
- **gRPC** — межсервисное общение
- **HTTP (REST)** — только между Frontend и API Gateway

## Архитектура

```
Frontend (Next.js) → HTTPS → API Gateway (Go, REST → gRPC) → сервисы по gRPC
```

### Сервисы

| Сервис | Директория | Назначение |
|--------|-----------|------------|
| API Gateway | `gateway/` | Единая точка входа, REST→gRPC, rate limiting |
| Auth Service | `auth/` | Регистрация, логин, JWT (RS256), сессии |
| TwoFA Service | `twofa/` | Оркестрация 2FA, Shamir split/combine, TOTP |
| MPC Node (x3) | `mpc/` | Хранение одной доли секрета (AES-256-GCM at-rest) |

### Потоки данных

- **Gateway → Auth**: auth operations, валидация токенов
- **Gateway → TwoFA**: 2FA operations (setup, verify, disable, status)
- **TwoFA → MPC nodes (x3)**: share operations (store, retrieve, delete)
- **Auth → Redis**: refresh-токены, сессии
- **Gateway → Redis**: rate limit counters
- **Все сервисы → PostgreSQL**: персистентные данные
- **Все сервисы → Kafka**: события аудита

## Структура каждого сервиса (Clean Architecture)

Каждый сервис — отдельный Go-модуль на верхнем уровне, по образцу medialog/students:

```
<service>/
├── api/                          # Proto-определения
│   ├── google/api/               # Google API annotations
│   ├── models/                   # Proto-модели данных
│   └── <service>_api/            # Proto-сервис (RPC-методы)
├── cmd/app/main.go               # Точка входа, graceful shutdown
├── config/config.go              # Загрузка конфигурации из config.yaml
├── internal/
│   ├── api/<service>_service_api/ # gRPC handlers (по одному файлу на метод)
│   ├── bootstrap/                # DI-фабрики для всех зависимостей
│   ├── models/models.go          # Доменные модели
│   ├── services/<serviceName>/   # Бизнес-логика (по одному файлу на метод)
│   ├── storage/pgstorage/        # PostgreSQL repository (pgx, без ORM)
│   ├── pb/                       # Сгенерированный protobuf-код
│   └── middleware/interceptors.go # gRPC interceptors
├── scripts/                      # generate.sh, command.mk
├── config.yaml
├── docker-compose.yaml
├── Makefile
├── go.mod
└── go.sum
```

## Правила разработки

### Общие

- **Clean Architecture**: handler → service → repository, зависимости через интерфейсы
- **DI через bootstrap**: каждая зависимость создается фабрикой в `internal/bootstrap/`
- **Конфигурация**: config.yaml, загрузка в `config/config.go` (gopkg.in/yaml.v3)
- **Логирование**: slog (structured), НИКОГДА не логировать секреты, пароли, доли, ключи шифрования
- **Ошибки**: gRPC status codes (InvalidArgument, NotFound, Unauthenticated, AlreadyExists, Internal)
- **Метрики**: Prometheus (github.com/prometheus/client_golang)
- **БД**: pgx напрямую, БЕЗ ORM (GORM и т.п.), инициализация таблиц через initTables
- **HTTP**: ТОЛЬКО в Gateway, все остальные сервисы — ТОЛЬКО gRPC
- **gRPC Health Check Protocol** в каждом сервисе
- **Graceful shutdown** с закрытием всех подключений

### Безопасность

- **Пароли**: bcrypt cost=12, валидация перед хешированием:
  - Минимум 12 символов
  - Минимум 1 строчная (a-z), 1 заглавная (A-Z), 1 цифра (0-9), 1 спецсимвол
  - Запрет 4+ символов подряд в последовательности (1234, abcd, qwer — и обратные)
- **JWT**: RS256, access 15 мин, refresh 7 дней (хранится в Redis с TTL)
- **TOTP-секрет**: НИКОГДА не персистируется целиком — только транзиентно в памяти, zeroize после использования
- **Shamir Secret Sharing**: 2-of-3, реализация с нуля в GF(256), НЕ сторонние библиотеки
- **Доли в MPC-нодах**: шифрование at-rest AES-256-GCM, nonce через crypto/rand
- **Rate limiting**: макс 5 попыток верификации 2FA за 5 минут на user_id
- **Kafka-аудит**: события содержат user_id, operation, timestamp — НИКОГДА не share_data или секреты

### Запреты

- НЕ добавлять фичи вне ТЗ (OAuth, SSO, email verification и т.п.)
- НЕ использовать ORM
- НЕ использовать сторонние библиотеки для Shamir
- НЕ хранить TOTP-секрет целиком ни в одном хранилище
- НЕ логировать секретные данные
- НЕ добавлять HTTP в сервисы кроме Gateway

## Зависимости Go

```
google.golang.org/grpc
github.com/jackc/pgx/v5
github.com/redis/go-redis/v9
github.com/segmentio/kafka-go
github.com/golang-jwt/jwt/v5
github.com/prometheus/client_golang
github.com/google/uuid
golang.org/x/crypto
gopkg.in/yaml.v3
```

## Obsidian (workspace/)

Корень проекта — Obsidian vault (`.obsidian/` в корне). Все заметки в `workspace/`. При работе над проектом:

- **Фиксируй решения** в `workspace/04 - Decisions/ADR Log.md` — каждое архитектурное или техническое решение
- **Обновляй прогресс** в `workspace/05 - Progress/TODO.md` — отмечай выполненные задачи
- **Веди changelog** в `workspace/05 - Progress/Changelog.md` — что сделано и когда
- **Документируй сервисы** в `workspace/02 - Services/` — при изменении API, добавлении эндпоинтов
- **Документируй безопасность** в `workspace/03 - Security/` — при изменении криптографических решений

### Структура vault
```
workspace/
├── 00 - Index.md                  # Главная навигация
├── 01 - Architecture/             # Архитектура, потоки данных
├── 02 - Services/                 # Документация по каждому сервису
├── 03 - Security/                 # Протоколы безопасности (Shamir, TOTP, AES, JWT, пароли)
├── 04 - Decisions/                # ADR — архитектурные решения
└── 05 - Progress/                 # TODO, Changelog
```

### Правила ведения заметок
- Используй Obsidian wikilinks: `[[имя заметки]]`
- Каждая заметка — один топик, не мешай всё в одну
- При добавлении нового сервиса/модуля — создавай заметку в соответствующей папке
- При принятии нетривиального решения — добавляй ADR с датой и обоснованием

## Вспомогательные директории

| Директория | Назначение |
|-----------|------------|
| `migration/` | Общие миграции (если нужны cross-service) |
| `monitoring/` | Конфигурация Prometheus + Grafana |
| `workspace/` | Заметки, решения, прогресс |

<!-- GSD:project-start source:PROJECT.md -->
## Project

**MPC-2FA — Двухфакторная аутентификация с распределенным хранением секретов**

Система двухфакторной аутентификации на микросервисной архитектуре с распределенным хранением TOTP-секретов через протокол Shamir Secret Sharing (2-of-3). -проект: Go-микросервисы, gRPC, Clean Architecture по образцу medialog/students. TOTP-секрет никогда не хранится целиком — разделяется на 3 доли, каждая хранится на отдельной MPC-ноде с шифрованием AES-256-GCM at-rest.

**Core Value:** TOTP-секрет никогда не существует в персистентном хранилище целиком — безопасность через распределение и шифрование долей.

### Constraints

- **Tech Stack**: Go 1.26.2, PostgreSQL, Redis 8.6.2, Kafka 4.1.2, gRPC, pgx (без ORM)
- **Go Modules**: `github.com/vbncursed/vkr/auth`, `github.com/vbncursed/vkr/twofa`, `github.com/vbncursed/vkr/mpc`
- **Security**: bcrypt cost=12, JWT RS256, AES-256-GCM, Shamir 2-of-3 в GF(256)
- **Architecture**: Clean Architecture (medialog/students), каждый сервис — отдельный Go-модуль
- **Academic**:  — все криптографические компоненты реализуются с нуля (Shamir, TOTP)
- **Logging**: НИКОГДА не логировать секреты, пароли, доли, ключи шифрования
<!-- GSD:project-end -->

<!-- GSD:stack-start source:codebase/STACK.md -->
## Technology Stack

## Languages
- Go 1.26.2 - All backend services (gateway, auth, twofa, mpc)
- Next.js - Web UI (separate repository, communicates via HTTPS REST)
## Runtime
- Go runtime 1.26.2
- Go modules (go.mod / go.sum per service)
## Frameworks
- gRPC (google.golang.org/grpc) - Inter-service communication between Gateway, Auth Service, TwoFA Service, and MPC Nodes
- Protocol Buffers (protobuf) - Service contracts and message definitions
- Google API Annotations - HTTP→gRPC transcoding in Gateway
- HTTP REST API - Single entry point from Next.js frontend
- Net/http standard library with gRPC-Gateway for REST→gRPC translation
## Key Dependencies
- `google.golang.org/grpc` - Core RPC framework for inter-service communication
- `github.com/jackc/pgx/v5` - PostgreSQL driver (raw SQL, no ORM)
- `github.com/redis/go-redis/v9` - Redis client for session storage and rate limiting
- `github.com/segmentio/kafka-go` - Kafka client for audit events
- `github.com/golang-jwt/jwt/v5` - JWT token generation and validation (RS256)
- `golang.org/x/crypto` - Cryptographic operations: bcrypt (password hashing), AES-256-GCM (share encryption)
- `github.com/prometheus/client_golang` - Prometheus metrics collection
- `github.com/google/uuid` - UUID generation for user and session IDs
- `gopkg.in/yaml.v3` - Configuration file parsing (config.yaml)
## Configuration
- `config.yaml` per service - Configuration loaded via `internal/config/config.go`
- Required config sections per service:
- Not explicitly documented, but config.yaml loading pattern suggests ENCRYPTION_KEY, NODE_ID (for MPC nodes)
- Makefile per service - Build, proto generation, Docker targets
- `scripts/generate.sh` per service - Protocol Buffer code generation
- `docker-compose.yaml` per service - Local development: PostgreSQL, Redis
## Service Structure
- HTTP REST endpoint for Next.js frontend
- gRPC clients to Auth and TwoFA services
- Rate limiting via Redis
- Request routing and authentication validation
- User registration and login
- JWT token generation (RS256: access 15m, refresh 7d)
- Password validation and bcrypt hashing (cost=12)
- Session management via Redis
- PostgreSQL user/session storage
- TOTP provisioning and verification
- Shamir Secret Sharing orchestration (2-of-3 split)
- gRPC clients to 3 MPC nodes for share distribution
- Backup code generation and storage
- Rate limiting for 2FA attempts (5 per 5 minutes per user)
- Share storage with AES-256-GCM encryption at rest
- gRPC endpoints for StoreShare, RetrieveShare, DeleteShare
- Per-node PostgreSQL database
## Data Storage
- Databases per service (or shared, depending on deployment)
- Tables:
- No ORM (pgx used directly)
- Initialization via `initTables()` in each service's storage layer
- Refresh token storage: `refresh_token:{token_jti}` → user_id with TTL 7 days
- Rate limit counters: `rate_limit:{user_id}:{operation}` with TTL 5 minutes
- Audit topics (one per service or unified):
- Events contain: user_id, operation, timestamp, node_id (mpc only)
- Never contains: passwords, TOTP secrets, share data, encryption keys
## Cryptography
- bcrypt, cost=12
- RS256 (RSA-2048 + SHA-256)
- Access token: 15 minutes
- Refresh token: 7 days (stored in Redis)
- Keys managed via config.yaml (private/public key paths)
- Shamir Secret Sharing (custom implementation, GF(256), 2-of-3)
- Implementation location: `twofa/internal/services/twofaService/shamir/`
- AES-256-GCM at rest in MPC nodes
- Nonce: 12 bytes, generated via crypto/rand per operation
- Key: 32 bytes from config.yaml (ENCRYPTION_KEY)
- RFC 6238 compliant
- 6-digit codes, 30-second window
- Validation: ±1 time window tolerance
- Secret: 20 bytes, base32 encoded
- Never persisted; only exists in memory during verify/setup
## Deployment
- Docker Compose files per service (PostgreSQL, Redis, Kafka)
- Service containerization via Dockerfile (not shown in current codebase state)
- Prometheus metrics: request counts, durations, errors per service
- Grafana dashboards (configuration in `monitoring/` directory)
- Structured logging: slog (standard library)
- Never log: passwords, TOTP secrets, share data, encryption keys
- All services: proper connection closure (PostgreSQL, Redis, Kafka, gRPC listeners)
- Health check: gRPC Health Check Protocol enabled on all services
<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->
## Conventions

## Naming Patterns
- **Handler files** (gRPC API): One method per file within `internal/api/<service>_service_api/`. Example: `register.go`, `login.go`, `refresh_token.go` (snake_case)
- **Service files** (Business logic): One method per file within `internal/services/<serviceName>/`. Example: `setup.go`, `verify.go`, `disable.go` (snake_case)
- **Storage files**: One entity type per file within `internal/storage/pgstorage/`. Example: `user.go`, `session.go`, `share.go` (snake_case)
- **Test files**: Append `_test.go` suffix to the implementation file (same directory). Example: `password_validation_test.go` in `internal/services/authService/`
- **Utility/Domain files**: Clear, single-responsibility naming. Example: `shamir.go`, `gf256.go`, `aes.go`, `totp.go` (snake_case)
- **Handler functions**: Receiver method on service struct. Example: `(api *AuthServiceAPI) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.TokenPair, error)`
- **Service methods**: Receiver method on service struct. Example: `(s *AuthService) ValidatePassword(password string) error`
- **Repository methods**: Receiver method on storage struct. Example: `(ps *PGStorage) CreateUser(ctx context.Context, user *User) error`
- **Internal helpers**: PascalCase (unexported). Example: `validateEmail`, `hashPassword`, `generateJWT`
- **Exported functions**: PascalCase. Example: `NewAuthService`, `NewPGStorage`, `Split`, `Combine`
- **Short-lived loop/temp variables**: Single letter (i, j, ctx, err). Example: `for i := 0; i < len(shares); i++`
- **Domain objects**: Clear descriptive names. Example: `user`, `share`, `token`, `secret`, `nonce`
- **Constants**: SCREAMING_SNAKE_CASE. Example: `MAX_ATTEMPTS`, `RATE_LIMIT_WINDOW`, `JWT_EXPIRY_MINUTES`, `COST_BCRYPT`
- **Interfaces**: Capitalized, descriptive. Example: `Storage`, `Service`, `Encryptor`, `Producer`
- **Error variables**: Prefixed with `err` or `Err`. Example: `errNotFound`, `ErrInvalidPassword`
- **Struct names**: PascalCase, nouns. Example: `User`, `AuthService`, `PGStorage`, `Share`, `TokenPair`
- **Interface names**: PascalCase, -er suffix for behavior. Example: `Encryptor`, `Producer`, `Validator`
- **Proto message names**: PascalCase, noun. Example: `RegisterRequest`, `User`, `TokenPair`
## Code Style
- **Tool**: gofmt (built-in Go formatter)
- **Indentation**: 1 tab = 8 spaces (Go standard)
- **Line length**: No hard limit, but keep under 120 characters for readability
- **Blank lines**: Use between logical sections within functions, between methods
- **Tool**: golangci-lint (or equivalent)
- **Key rules enforced**:
- **Linting enforced**: Every exported function/type must have a comment
- **Format**: Start with the exported name. Example:
- **Unexported helpers**: Optional but recommended. Example:
- **Complex logic blocks**: Inline comments explain intent, not what code does. Example:
## Import Organization
- **No aliases** — use full import paths with package prefixes
- **Local packages**: Import as `auth/internal/models` (not aliased)
- **Example**:
## Error Handling
- **Explicit error checks**: Always check returned errors immediately
- **Wrapping errors**: Use `%w` with `fmt.Errorf` to preserve error chain
- **gRPC errors**: Convert domain errors to gRPC status codes in handlers
## Logging
- **Structured logging only**: All logs use key-value pairs
- **Secret data NEVER logged**: No passwords, TOTP secrets, share data, encryption keys, JWT tokens
- **Error logging**: Always include error with context
- **Metric context**: Include observable metadata (user_id, operation, status)
## Dependency Injection
## Configuration
## Repository (Storage) Pattern
- One method per CRUD operation or query
- Always accept `ctx context.Context` as first parameter
- Return errors directly (convert to gRPC codes in handler layer)
- Use parameterized queries ($1, $2, etc.) to prevent SQL injection
## Handler (gRPC API) Pattern
- Minimal validation in handler (basic checks only)
- Delegate business logic to service layer
- Convert errors to appropriate gRPC status codes
- No detailed error messages returned to client for security (log internally instead)
## Service (Business Logic) Pattern
- Contains all business logic and validation
- Uses repository for data access
- Returns domain errors (not gRPC codes)
- Publishes events asynchronously (failures don't block main operation)
## Crypto and Security
- Algorithm: bcrypt with cost=12
- Always pre-validate password before hashing (see `internal/services/authService/password_validation.go`)
- Example:
- Private key: Held only by Auth Service
- Public key: Distributed to Gateway and other services for verification
- Claims structure:
- Expiry: Access 15 minutes, Refresh 7 days
- NEVER persist whole secrets
- After split/combine in memory → immediately zeroize using `subtle.ConstantTimeCompare` or manual byte clearing
- Example:
- Unique nonce per operation: `crypto/rand` (12 bytes)
- Encryption key: From config (NODE_ID specific)
- Example in MPC Node:
## Function Design
- Aim for functions under 30 lines
- Extract helper functions for complex logic blocks
- Each function should have a single responsibility
- Max 3-4 parameters; use struct for related params
- Always include `ctx context.Context` as first parameter for handlers/services
- Example:
- Always return errors as last value: `(result Type, error)`
- Prefer `(value, error)` over `error` only
- Never ignore returned errors in production code
## Module Design
- Capitalize only what's meant for external use
- Private types/functions use lowercase (package-internal)
- Example:
- Each file imports directly from subpackages
- Example: `import "auth/internal/services/authService"` (not from `auth/internal/services/`)
## Middleware and Interceptors
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->
## Architecture

## Pattern Overview
- API Gateway as single HTTP entry point, all inter-service communication via gRPC
- Distributed secret storage using Shamir Secret Sharing (2-of-3 threshold)
- Event-driven audit logging through Kafka
- Clear separation of concerns: Authentication, 2FA orchestration, MPC node storage
- Dependency Injection via bootstrap factories
## Layers
- Purpose: Handle incoming gRPC RPC calls, validate requests, coordinate with service layer
- Location: `<service>/internal/api/<service>_service_api/`
- Contains: One file per RPC method (e.g., `register.go`, `login.go`, `setup.go`)
- Depends on: Service layer (business logic), protobuf models
- Used by: gRPC clients (Gateway, other services)
- Purpose: Implement domain logic, orchestrate operations, manage state transitions
- Location: `<service>/internal/services/<serviceName>/`
- Contains: One file per major operation (e.g., `register.go`, `login.go`, `password_validation.go`)
- Depends on: Repository layer (data access), external services (Redis, Kafka)
- Used by: API layer (handlers)
- Purpose: Abstract database and cache access, provide persistence
- Location: `<service>/internal/storage/pgstorage/` and `<service>/internal/storage/redisstorage/`
- Contains: PostgreSQL repositories using pgx directly (no ORM), Redis session management
- Depends on: Database drivers (pgx v5, redis/go-redis)
- Used by: Service layer
- Purpose: Create and wire all service dependencies, manage component lifecycle
- Location: `<service>/internal/bootstrap/`
- Contains: Factory functions for each major component (services, storage, clients, server)
- Depends on: All other layers
- Used by: `main()` in `cmd/app/main.go`
- Purpose: Cross-cutting concerns: authentication, rate limiting, logging, metrics, error recovery
- Location: `<service>/internal/middleware/interceptors.go`
- Contains: gRPC unary and stream interceptors
- Depends on: Service layer (for validation), logging (slog), metrics (Prometheus)
- Used by: gRPC server configuration in bootstrap
## Data Flow
- **JWT Tokens**: Generated by Auth Service (RS256), stored client-side, validated on each gRPC call
- **Refresh Tokens**: Stored in Redis with TTL, used to rotate JWT pairs
- **TOTP Secrets**: Never persisted in plaintext — reconstructed on-demand from encrypted shares
- **Shares**: Stored encrypted at-rest in MPC node databases, retrieved only when needed for verification
- **Rate Limits**: Counters stored in Redis (Gateway for general, TwoFA for 2FA attempts)
- **Audit Events**: Immutable log in PostgreSQL + real-time stream to Kafka
## Key Abstractions
- Purpose: Encapsulates user authentication and JWT management
- Examples: `auth/internal/services/authService/auth_service.go`, `register.go`, `login.go`, `jwt.go`
- Pattern: Dependency-injected service with methods for each operation (Register, Login, RefreshToken, Logout, ValidateToken)
- Purpose: Orchestrates 2FA setup, verification, and disable operations with MPC node coordination
- Examples: `twofa/internal/services/twofaService/twofa_service.go`, `setup.go`, `verify.go`
- Pattern: Coordinates multiple gRPC calls to MPC nodes, manages Shamir split/combine lifecycle
- Purpose: Splits and reconstructs secrets using Shamir Secret Sharing (2-of-3)
- Examples: `twofa/internal/services/twofaService/shamir/shamir.go`
- Pattern: Custom implementation in GF(256) with Lagrange interpolation
- Purpose: Generates and validates TOTP codes per RFC 6238
- Examples: `twofa/internal/services/twofaService/totp/totp.go`
- Pattern: Time-based one-time password with 30-second windows
- Purpose: Abstract database access using pgx directly (no ORM)
- Examples: `<service>/internal/storage/pgstorage/pgstorage.go`, `user.go`, `session.go`
- Pattern: Separate file per entity, constructor with connection pool, methods for CRUD
- Purpose: Manage session tokens and rate limit counters with TTL
- Examples: `<service>/internal/storage/redisstorage/redisstorage.go`, `session.go`
- Pattern: Wrapper around redis/go-redis client with TTL management
- Purpose: Store, retrieve, and delete encrypted shares with access control
- Examples: `mpc/internal/services/mpcService/mpc_service.go`, `store_share.go`, `retrieve_share.go`
- Pattern: Delegates encryption/decryption to storage layer, enforces shared-secret authorization
## Entry Points
- Location: `gateway/cmd/app/main.go`
- Triggers: Process startup, listens on HTTP port (default 8080)
- Responsibilities:
- Location: `auth/cmd/app/main.go`
- Triggers: Process startup, listens on gRPC port (default 9090)
- Responsibilities:
- Location: `twofa/cmd/app/main.go`
- Triggers: Process startup, listens on gRPC port (default 9091)
- Responsibilities:
- Location: `mpc/cmd/app/main.go`
- Triggers: Process startup, listens on gRPC port (configurable per node)
- Responsibilities:
## Error Handling
- `InvalidArgument`: Validation failures (bad email format, weak password, invalid OTP)
- `NotFound`: Resource not found (user, 2FA record, share)
- `AlreadyExists`: Duplicate creation attempts (email already registered, 2FA already enabled)
- `Unauthenticated`: Token validation failures, authorization failures
- `PermissionDenied`: User doesn't have permission to access resource
- `FailedPrecondition`: Operation preconditions not met (2FA not enabled for disable, share count < 2 for combine)
- `Internal`: Unexpected errors (database errors, encryption errors)
## Cross-Cutting Concerns
- **Never log:** passwords, TOTP secrets, encryption keys, share data
- **Always log:** user_id, operation name, error type, timing information
- Implementation: gRPC unary/stream interceptors in `<service>/internal/middleware/interceptors.go`
- Auth Service: password policy (length, character classes, no sequences)
- TwoFA Service: OTP format, rate limiting
- All handlers: required field checks, email format validation
- Implementation: Validator functions in service layer before persistence
- JWT (RS256) with 15-minute access token expiry
- Refresh token rotation with 7-day TTL in Redis
- gRPC metadata inspection for Authorization header
- Implementation: Auth Service ValidateToken method, interceptor for token validation
- Gateway: per-IP general rate limiting
- TwoFA: per-user 2FA verification attempts (5 per 5 minutes)
- Implementation: Redis counters with Lua scripts or simple increment/expire
- All operations published to Kafka: `<domain>.<operation>` topics
- Payload: user_id, operation, timestamp, status (never secret data)
- Consumer: Separate audit service (future) or log sink
- `auth_requests_total{method, status}`: Auth operation counters
- `auth_request_duration_seconds`: Auth latency histogram
- `twofa_operations_total{operation, status}`: TwoFA operation counters
- `twofa_mpc_latency_seconds{node_id}`: MPC node request latency
- `mpc_operations_total{node_id, operation, status}`: MPC node operation counters
- Implementation: Interceptor middleware wrapping handler calls
<!-- GSD:architecture-end -->

<!-- GSD:skills-start source:skills/ -->
## Project Skills

No project skills found. Add skills to any of: `.claude/skills/`, `.agents/skills/`, `.cursor/skills/`, or `.github/skills/` with a `SKILL.md` index file.
<!-- GSD:skills-end -->

<!-- GSD:workflow-start source:GSD defaults -->
## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:
- `/gsd-quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd-debug` for investigation and bug fixing
- `/gsd-execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->

<!-- GSD:profile-start -->
## Developer Profile

> Profile not yet configured. Run `/gsd-profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
