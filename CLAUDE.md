# MPC-2FA — Двухфакторная аутентификация с распределенным хранением секретов

## Проект

: Разработка двухфакторной системы аутентификации с распределенным хранением секретов и использованием протоколов безопасных многосторонних вычислений.

**Core Value:** TOTP-секрет никогда не существует в персистентном хранилище целиком — безопасность через распределение и шифрование долей.

## Стек

- **Go** 1.26.2
- **PostgreSQL** — users, sessions, audit, 2FA metadata, shares
- **Redis** 8.6.2 — refresh-сессии (Auth), rate limiting и OTP-счетчики (TwoFA)
- **Kafka** 4.1.2 — события аудита между сервисами
- **Prometheus** + **Grafana** — мониторинг и метрики
- **gRPC** — межсервисное общение
- **HTTP (REST)** — только между Frontend и API Gateway

## Архитектура

```
Frontend (Next.js) → HTTPS → API Gateway (Go, REST → gRPC) → сервисы по gRPC
```

### Сервисы

| Сервис | Директория | Статус | Назначение |
|--------|-----------|--------|------------|
| API Gateway | `gateway/` | Не реализован | Единая точка входа, REST→gRPC, rate limiting |
| Auth Service | `auth/` | Реализован | Регистрация, логин, JWT (RS256), сессии |
| TwoFA Service | `twofa/` | Реализован | Оркестрация 2FA, Shamir split/combine, TOTP |
| MPC Node (x3) | `mpc/` | Реализован | Хранение одной доли секрета (AES-256-GCM at-rest) |

### Потоки данных

- **Gateway → Auth**: auth operations, валидация токенов
- **Gateway → TwoFA**: 2FA operations (setup, verify, disable, status)
- **TwoFA → MPC nodes (x3)**: share operations (store, retrieve, delete)
- **Auth → Redis**: refresh-токены, сессии (token families)
- **TwoFA → Redis**: rate limit counters, OTP-счетчики
- **Gateway → Redis**: rate limit counters (будущее)
- **Все сервисы → PostgreSQL**: персистентные данные
- **Все сервисы → Kafka**: события аудита

## Структура каждого сервиса (Clean Architecture)

Каждый сервис — отдельный Go-модуль на верхнем уровне:

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
│   ├── domain/                    # Доменные модели и ошибки
│   ├── services/<serviceName>/   # Бизнес-логика (по одному файлу на метод)
│   ├── storage/pgstorage/        # PostgreSQL repository (pgx, без ORM)
│   ├── storage/redisstorage/     # Redis storage (Auth: сессии, TwoFA: rate limiting)
│   ├── crypto/                   # Криптографические модули (только TwoFA: shamir/, totp/, zeroize)
│   ├── pb/                       # Сгенерированный protobuf-код
│   └── middleware/               # gRPC interceptors (interceptors.go, metrics.go)
├── scripts/                      # generate.sh, command.mk
├── config.yaml
├── docker-compose.yaml
├── Makefile
├── go.mod
└── go.sum
```

### Отличия между сервисами

| Аспект | Auth | TwoFA | MPC |
|--------|------|-------|-----|
| Доменные модели | `internal/domain/` (models.go, errors.go) | `internal/domain/` (models.go, errors.go) | `internal/domain/` (models.go, errors.go) |
| Redis | `storage/redisstorage/` (сессии) | `storage/redisstorage/` (rate limit, OTP, noop) | Нет Redis |
| Криптография | В сервисном слое (jwt.go, password_validation.go) | `internal/crypto/` (shamir/, totp/, zeroize.go) | В сервисном слое (encrypt.go) |
| Bootstrap | Отдельные файлы (auth_service.go, pgstorage.go, ...) | Консолидированный bootstrap.go + kafka.go + mpc_adapter.go | Консолидированный bootstrap.go + kafka.go |
| Моки | `services/authService/mocks/` (minimock) | `services/twofaService/mocks/` (minimock) | `services/mpcService/mocks/` (minimock) |

## gRPC API

### Auth Service (6 RPC)
| RPC | Описание |
|-----|----------|
| Register | Регистрация по email + password |
| Login | Логин, выдача access + refresh токенов |
| RefreshToken | Ротация refresh-токена |
| Logout | Удаление refresh-токена, инвалидация сессии |
| LogoutAll | Удаление ВСЕХ refresh-токенов пользователя |
| ValidateToken | Проверка access-токена, возврат user_id + claims |

### TwoFA Service (4 RPC)
| RPC | Описание |
|-----|----------|
| Setup2FA | Генерация секрета → Shamir split → MPC → provisioning URI + backup-коды |
| Verify2FA | Запрос 2 долей → Shamir combine → TOTP-проверка → zeroize |
| Disable2FA | Верификация OTP → удаление долей и метаданных |
| Get2FAStatus | Статус 2FA (is_enabled, created_at) |

### MPC Node Service (3 RPC)
| RPC | Описание |
|-----|----------|
| StoreShare | Шифрование AES-256-GCM → сохранение в PostgreSQL |
| RetrieveShare | Чтение → дешифрование → возврат |
| DeleteShare | Удаление всех долей пользователя |

## Правила разработки

### Общие

- **Clean Architecture**: handler → service → repository, зависимости через интерфейсы
- **DI через bootstrap**: каждая зависимость создается фабрикой в `internal/bootstrap/`
- **Конфигурация**: config.yaml, загрузка в `config/config.go` (go.yaml.in/yaml/v4)
- **Логирование**: slog (structured), НИКОГДА не логировать секреты, пароли, доли, ключи шифрования
- **Ошибки**: gRPC status codes (InvalidArgument, NotFound, Unauthenticated, AlreadyExists, FailedPrecondition, Internal)
- **Метрики**: Prometheus (github.com/prometheus/client_golang), отдельный metrics_port
- **БД**: pgx напрямую, БЕЗ ORM (GORM и т.п.), инициализация таблиц через initTables
- **HTTP**: ТОЛЬКО в Gateway, все остальные сервисы — ТОЛЬКО gRPC
- **gRPC Health Check Protocol** в каждом сервисе
- **Graceful shutdown** с закрытием всех подключений (30s timeout)
- **Тестирование**: minimock для моков, gotest.tools/v3 для ассертов, table-driven тесты

### Безопасность

- **Пароли**: bcrypt cost=12, валидация перед хешированием:
  - Минимум 12 символов
  - Минимум 1 строчная (a-z), 1 заглавная (A-Z), 1 цифра (0-9), 1 спецсимвол
  - Запрет 4+ символов подряд в последовательности (1234, abcd, qwer — и обратные)
- **JWT**: RS256, access 15 мин, refresh 7 дней (хранится в Redis с TTL)
  - Token families для обнаружения повторного использования refresh-токенов
  - Lua-скрипт для атомарного удаления всех токенов пользователя
- **TOTP-секрет**: НИКОГДА не персистируется целиком — только транзиентно в памяти, zeroize после использования (`crypto/zeroize.go`)
- **Shamir Secret Sharing**: 2-of-3, реализация с нуля в GF(256), НЕ сторонние библиотеки
  - Расположение: `twofa/internal/crypto/shamir/` (shamir.go, gf256.go)
- **TOTP**: RFC 6238, реализация с нуля
  - Расположение: `twofa/internal/crypto/totp/` (totp.go, uri.go)
- **Доли в MPC-нодах**: шифрование at-rest AES-256-GCM, nonce через crypto/rand
  - Шифрование в: `mpc/internal/services/mpcService/encrypt.go`
- **Rate limiting**: макс 5 попыток верификации 2FA за 5 минут на user_id (Redis)
- **MPC авторизация**: shared secret через gRPC metadata, проверка в interceptor
- **Kafka-аудит**: события содержат user_id, operation, timestamp — НИКОГДА не share_data или секреты
- **Backup-коды**: 10 штук, хеши bcrypt, verify_backup_code.go

### Запреты

- НЕ добавлять фичи вне ТЗ (OAuth, SSO, email verification и т.п.)
- НЕ использовать ORM
- НЕ использовать сторонние библиотеки для Shamir
- НЕ хранить TOTP-секрет целиком ни в одном хранилище
- НЕ логировать секретные данные
- НЕ добавлять HTTP в сервисы кроме Gateway

## Зависимости Go

### Общие (все сервисы)
```
google.golang.org/grpc v1.80.0
google.golang.org/protobuf v1.36.11
github.com/jackc/pgx/v5 v5.9.1
github.com/segmentio/kafka-go v0.4.50
github.com/prometheus/client_golang v1.23.2
github.com/google/uuid v1.6.0
go.yaml.in/yaml/v4 v4.0.0-rc.4
```

### Auth
```
github.com/golang-jwt/jwt/v5 v5.3.1
github.com/redis/go-redis/v9 v9.18.0
golang.org/x/crypto v0.50.0
```

### TwoFA
```
github.com/redis/go-redis/v9 v9.18.0
golang.org/x/crypto v0.50.0
golang.org/x/sync v0.20.0          # errgroup для параллельных вызовов MPC
```
- `replace github.com/vbncursed/vkr/mpc => ../mpc` (локальная зависимость на proto MPC)

### MPC
- Без Redis, без x/crypto (шифрование через стандартный crypto/aes + crypto/cipher)

### Тестирование (все сервисы)
```
github.com/gojuno/minimock/v3 v3.4.7
gotest.tools/v3 v3.5.2
```

## Конфигурация (config.yaml)

### Auth
```yaml
server: { port, metrics_port, log_level }
database: { dsn }
redis: { addr, password, db }
kafka: { brokers, topic }
jwt: { private_key_path, public_key_path, access_token_ttl, refresh_token_ttl }
```

### TwoFA
```yaml
server: { port, metrics_port, log_level }
database: { dsn }
redis: { addr, password, db }
kafka: { brokers, topic }
mpc_nodes: ["addr1", "addr2", "addr3"]  # ровно 3 ноды
shared_secret: "..."                      # для авторизации на MPC-нодах
mpc_timeout: 5s                           # default
```
Env overrides: `TWOFA_SHARED_SECRET`, `TWOFA_DATABASE_DSN`

### MPC
```yaml
server: { port, metrics_port, log_level }
database: { dsn }
kafka: { brokers, topic }
node: { id, encryption_key }             # уникальные для каждой ноды
shared_secret: "..."                      # для авторизации входящих запросов
```

## Redis Key Patterns

### Auth (redisstorage/session.go)
```
refresh_token:{jti}      → JSON(RefreshTokenData) с TTL
token_family:{family}    → SET of JTIs
user_tokens:{userID}     → SET of token families
```
- Pipeline для атомарных операций
- Lua-скрипт `deleteAllScript` для атомарного удаления всех токенов пользователя

### TwoFA (redisstorage/)
```
rate_limit:{user_id}:{operation}  → counter с TTL 5 мин (rate_limit.go)
otp_counter:{user_id}             → counter (otp_counter.go)
```
- `noop.go` — NoOp-реализация для fallback при недоступности Redis
- `cleanup.go` — очистка данных пользователя

## Kafka-события

| Сервис | Топик | Данные |
|--------|-------|--------|
| Auth | user.registered | user_id, operation, timestamp |
| Auth | user.logged_in | user_id, operation, timestamp |
| Auth | token.refreshed | user_id, operation, timestamp |
| TwoFA | 2fa.setup | user_id, operation, timestamp |
| TwoFA | 2fa.verified | user_id, operation, timestamp |
| TwoFA | 2fa.disabled | user_id, operation, timestamp |
| MPC | share.stored | user_id, share_index, node_id, timestamp |
| MPC | share.retrieved | user_id, share_index, node_id, timestamp |
| MPC | share.deleted | user_id, node_id, timestamp |

НИКОГДА не включать: passwords, TOTP secrets, share data, encryption keys.

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

### Принятые ADR
- ADR-001: Shamir для распределенного хранения секретов (vs. прямое хранение TOTP)
- ADR-002: Собственная реализация Shamir в GF(256) (vs. библиотеки)
- ADR-003: AES-256-GCM для шифрования долей at-rest
- ADR-004: RS256 JWT (vs. HS256) — асимметричная подпись для децентрализованной верификации
- ADR-005: Clean Architecture (handler → service → storage)
- ADR-006: pgx без ORM
- ADR-007: Миграция yaml.v3 → yaml/v4 и модернизация Go 1.26.2
- ADR-010: Rename internal/models to internal/domain in MPC and TwoFA

### Правила ведения заметок
- Используй Obsidian wikilinks: `[[имя заметки]]`
- Каждая заметка — один топик, не мешай всё в одну
- При добавлении нового сервиса/модуля — создавай заметку в соответствующей папке
- При принятии нетривиального решения — добавляй ADR с датой и обоснованием

## Вспомогательные директории

| Директория | Назначение | Статус |
|-----------|------------|--------|
| `gateway/` | API Gateway (REST→gRPC) | Пустая, не реализован |
| `migration/` | Общие миграции (если нужны cross-service) | Пустая |
| `monitoring/` | Конфигурация Prometheus + Grafana | Пустая |
| `workspace/` | Заметки, решения, прогресс | Активная |
| `.planning/` | GSD planning artifacts | Активная |

## Текущий статус реализации

### Реализовано
- Auth Service — полностью (register, login, JWT, refresh, validate, logout, logout_all, password validation, audit)
- TwoFA Service — полностью (setup, verify, disable, status, backup codes, rate limiting, verify_backup_code)
- MPC Node Service — полностью (store, retrieve, delete, AES-256-GCM, shared secret auth)
- Shamir Secret Sharing — custom GF(256), 2-of-3
- TOTP — RFC 6238, ±1 window, provisioning URI
- Unit-тесты для всех криптографических компонентов и бизнес-логики
- Kafka audit events
- Prometheus metrics

### Не реализовано
- API Gateway (`gateway/`) — отложен
- Docker Compose полной системы (есть только per-service compose)
- Prometheus scrape config + Grafana dashboards
- Интеграционные тесты
- Система миграций БД (таблицы создаются через initTables)

## Conventions

### Naming Patterns
- **Handler files** (gRPC API): snake_case, один файл на метод в `internal/api/<service>_service_api/`
- **Service files**: snake_case, один файл на метод в `internal/services/<serviceName>/`
- **Storage files**: snake_case, один файл на сущность в `internal/storage/pgstorage/`
- **Test files**: `*_test.go` рядом с реализацией
- **Mocks**: `internal/services/<serviceName>/mocks/` (minimock, `*_mock.go`)

### Naming Conventions
- **Exported functions**: PascalCase (`NewAuthService`, `Split`, `Combine`)
- **Unexported helpers**: camelCase (`validateEmail`, `hashPassword`, `generateJWT`)
- **Struct names**: PascalCase, nouns (`User`, `AuthService`, `PGStorage`, `Share`)
- **Interface names**: PascalCase, -er suffix (`Encryptor`, `Producer`, `Storage`)
- **Constants**: SCREAMING_SNAKE_CASE (`MAX_ATTEMPTS`, `COST_BCRYPT`)
- **Error variables**: `Err` prefix (`ErrInvalidPassword`, `ErrDuplicateEmail`)
- **Proto messages**: PascalCase (`RegisterRequest`, `TokenPair`)

### Code Style
- **Formatter**: gofmt
- **Line length**: до 120 символов
- **Import groups**: stdlib → external → internal (без алиасов)
- **Error handling**: explicit checks, `%w` для wrapping, gRPC status codes в handler-слое
- **Logging**: slog structured, key-value pairs, НИКОГДА не секреты
- **Functions**: до 30 строк, single responsibility, `ctx context.Context` первым параметром
- **Returns**: `(result Type, error)`, никогда не игнорировать ошибки

### Patterns
- **Handler**: минимальная валидация → делегирование в service → конвертация ошибок в gRPC codes
- **Service**: вся бизнес-логика, domain errors (не gRPC), async Kafka events
- **Repository**: один метод на CRUD, `ctx` первым, parameterized queries ($1, $2)
- **Bootstrap**: фабрики для каждого компонента, lifecycle management
- **Interceptors**: logging + metrics + recovery (в middleware/)

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
