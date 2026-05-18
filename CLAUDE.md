# MPC-2FA — Двухфакторная аутентификация с распределенным хранением секретов

## Проект

Разработка двухфакторной системы аутентификации с распределенным хранением секретов и использованием протоколов безопасных многосторонних вычислений.

**Core Value:** TOTP-секрет никогда не существует в персистентном хранилище целиком — безопасность через распределение и шифрование долей.

## Стек

- **Go** 1.26.3
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

## Документация проекта (`docs/`)

Каталог `docs/` — публичная документация проекта (commit'ится в репозиторий, в отличие от gitignored `.obsidian/`). Структура:

```
docs/
├── README.md                      # Хаб навигации (заменяет старый "00 - Index.md")
├── 01 - Architecture/             # Архитектура, потоки данных
├── 02 - Services/                 # Документация по каждому сервису + аудиты
├── 03 - Security/                 # Протоколы безопасности (Shamir, TOTP, AES, JWT, mTLS)
├── 04 - Decisions/                # ADR — архитектурные решения
└── 05 - Progress/                 # TODO, Changelog
```

Markdown-ссылки используют относительные пути с URL-encoding (`[name](path%20with%20spaces.md)`) — рендерятся корректно в GitHub. Wikilink-стиль `[[X]]` НЕ используется (несовместимо с GitHub).

### Обязательные действия после каждой работы (Post-Work Checklist)

После **каждого** завершённого блока работы (коммит, фикс, рефакторинг, новая фича) — обновить документацию:

| После какой работы | Что обновить | Где |
|---|---|---|
| Любой код-коммит | Changelog entry: дата, категория (fix/feature/refactor), описание | `docs/05 - Progress/Changelog.md` |
| Баг-фикс / рефакторинг | Отметка `[x]` в TODO + changelog entry | `docs/05 - Progress/TODO.md` + `Changelog.md` |
| Новый ADR или изменение подхода | ADR entry: дата, статус, контекст, решение, причина, последствия | `docs/04 - Decisions/ADR Log.md` |
| Изменение API сервиса (RPC, proto) | Обновить заметку сервиса: методы, зависимости, ссылки | `docs/02 - Services/<Service>.md` |
| Изменение security-логики | Обновить security-заметку: параметры, где используется | `docs/03 - Security/<Topic>.md` |
| Аудит / code review | Создать или обновить audit-заметку с планом фиксов | `docs/02 - Services/<Service> - Audit.md` |
| Новая заметка любого типа | Добавить ссылку в `docs/README.md` | `docs/README.md` |

**Это НЕ опционально.** Работа без обновления документации считается незавершённой.

### Правила связности графа (Graph Connectivity)

- **Минимум связей**: каждая заметка ОБЯЗАНА иметь минимум 1 входящую `[[ссылку]]` из другой заметки и 1 исходящую `[[ссылку]]` на другую заметку
- **Index — хаб**: `00 - Index.md` содержит ссылки на ВСЕ заметки vault. При создании новой заметки — добавить в Index
- **Orphan-заметки запрещены**: если заметка не связана ни с чем — связать с ближайшим контекстом или удалить
- **Wikilinks**: использовать формат `[[имя заметки]]` или `[[путь/имя заметки|отображаемый текст]]`
- **Cross-linking**: сервисные заметки ссылаются на security-заметки и наоборот. ADR ссылаются на затронутые сервисы

### Шаблоны заметок (Note Templates)

**Service note** (`02 - Services/<Name>.md`):
```markdown
# <Service Name>
## Назначение
## API (RPC-методы)
| RPC | Описание |
## Зависимости
- Инфра: [[PostgreSQL]], [[Redis]], [[Kafka]]
- Сервисы: [[другой сервис]]
## Security
- Ссылки: [[протокол безопасности]]
## Связанные решения
- [[ADR Log#ADR-NNN]]
```

**Security note** (`03 - Security/<Topic>.md`):
```markdown
# <Protocol/Mechanism>
## Описание
## Параметры
## Где используется
- [[сервис, который использует]]
## Связанные решения
- [[ADR Log#ADR-NNN]]
```

**ADR entry** (в `04 - Decisions/ADR Log.md`):
```markdown
## ADR-NNN: <Название>
- **Дата**: YYYY-MM-DD
- **Статус**: Accepted / Superseded / Deprecated
- **Контекст**: Почему возник вопрос
- **Решение**: Что решили
- **Причина**: Почему именно так
- **Последствия**: Что это влечёт
- **Связано**: [[сервис]], [[security-заметка]]
```

**Changelog entry** (в `05 - Progress/Changelog.md`):
```markdown
## YYYY-MM-DD
- **[fix]** Описание фикса — [[затронутый сервис]]
- **[feature]** Описание фичи — [[затронутый сервис]], [[ADR Log#ADR-NNN]]
- **[refactor]** Описание рефакторинга — [[затронутый сервис]]
```

### Принятые ADR
- ADR-001: Shamir для распределенного хранения секретов (vs. прямое хранение TOTP)
- ADR-002: Собственная реализация Shamir в GF(256) (vs. библиотеки)
- ADR-003: AES-256-GCM для шифрования долей at-rest
- ADR-004: RS256 JWT (vs. HS256) — асимметричная подпись для децентрализованной верификации
- ADR-005: Clean Architecture (handler → service → storage)
- ADR-006: pgx без ORM
- ADR-007: Миграция yaml.v3 → yaml/v4 и модернизация Go 1.26.3
- ADR-008: Bootstrap — одна фабрика на файл (per-component pattern)
- ADR-009: pgxpool — явная конфигурация пула соединений
- ADR-010: Rename internal/models to internal/domain in MPC and TwoFA

### Правила ведения заметок
- Используй Obsidian wikilinks: `[[имя заметки]]`
- Каждая заметка — один топик, не мешай всё в одну
- При добавлении нового сервиса/модуля — создавай заметку в соответствующей папке
- При принятии нетривиального решения — добавляй ADR с датой и обоснованием
- **Один топик — одна заметка**, не смешивать разные темы в одном файле
- **Язык**: русский для текста, английский для кода и имён сущностей

## Вспомогательные директории

| Директория | Назначение | Статус |
|-----------|------------|--------|
| `gateway/` | API Gateway (REST→gRPC) | Реализован (см. README) |
| `migration/` | Общие миграции (если нужны cross-service) | Пустая |
| `monitoring/` | Конфигурация Prometheus + Grafana | Активная |
| `docs/` | ADR, security deep-dives, changelog (commit'ится в репо) | Активная |
| `certs/` | Dev PKI (gitignored), генерируется `scripts/gen-certs.sh` | Активная |
| `.planning/` | GSD planning artifacts (gitignored) | Локальная |

## Текущий статус реализации

### Реализовано
- Auth Service — полностью (register, login, JWT, refresh, validate, logout, logout_all, password validation, audit)
- TwoFA Service — полностью (setup, verify, disable, status, backup codes, rate limiting, verify_backup_code)
- MPC Node Service — полностью (store, retrieve, delete, AES-256-GCM, shared secret auth)
- API Gateway — полностью (grpc-gateway, REST→gRPC, middleware-стек, ScalarUI, Docker)
- Frontend — Next.js 16 + Liquid Glass design system, light/dark, ru/en, 11 UI / 9 widgets, протекшн middleware, refresh-токены в httpOnly cookie
- Shamir Secret Sharing — custom GF(256), 2-of-3
- TOTP — RFC 6238, ±1 window, provisioning URI
- Unit-тесты для всех криптографических компонентов и бизнес-логики
- **MPC Fault Tolerance тесты (2026-05-10)** — 4 файла в `twofa/internal/services/twofaService/fault_tolerance_*_test.go`, 6 функций / 12 субтестов: 1 нода вниз → ОК, медленная нода → first-2-wins, timeout-сценарии, all-or-nothing для setup/disable
- Kafka audit events
- Prometheus metrics + Grafana dashboard (`mpc-2fa-overview.json`)

### Не реализовано
- Система миграций БД (таблицы создаются через initTables) — отложено
- **mTLS между сервисами** — Phase B (см. ниже)
- **Подпись служебных запросов** — Phase B (комментарий `SECURITY(WR-03): deferred to Phase 9` в коде)
- **Нагрузочное тестирование** (k6/wrk/Locust + отчёт + scaling recs) — Phase C
- **Frontend monitoring page** (страница со статусами сервисов / RPS / latency / error rate из Prometheus) — Phase D

## План закрытия отсутствующих пунктов ТЗ

Контекст: команда (Frontend + Backend) выявила недостающие пункты ТЗ. План закрытия согласован 2026-05-10.

| Phase | Описание | Зона | Status |
|-------|----------|------|--------|
| A | MPC fault tolerance — тесты 2-of-3 threshold модели + документация | Backend | ✅ Done (2026-05-10) |
| B | mTLS между всеми сервисами + ADR-011 (Path A: shared_secret сохранён как defense-in-depth) | Backend | ✅ Done (2026-05-10) |
| C | k6 нагрузочные тесты (login / setup / verify / mixed) + REPORT.md с p50/p95/p99 + рекомендации по масштабированию | Backend | ✅ Done (2026-05-10) |
| D | Frontend Monitoring page — Gateway `/admin/monitoring/snapshot` + Next.js страница с виджетами (ThroughputOverview, MpcNodeStatus, ServiceHealthGrid) | Frontend | ✅ Done (2026-05-10) |

### Phase A — что использовано

- **Тесты:** gotest.tools/v3/assert, minimock/v3, переиспользуются существующие suites (`verifySuite`, `setupSuite`, `disableSuite`)
- **Helper:** `newVerifySuiteWithTimeout(t, d)` для timeout-сценариев с MPCTimeout=200ms
- **Сценарии:** OneNodeDown_Succeeds (×3 ноды), SlowNodeIgnored_FirstTwoWins, OneNodeTimeout_OneNodeDown_Fails, AllNodesTimeout_Fails, Setup_OneNodeDown_AllOrNothing (×3), Disable_OneNodeDown_PreservesRecord (×3)
- **Doc:** `docs/03 - Security/MPC Fault Tolerance.md` — описание threshold модели, поведения по флоу, известных ограничений
- **Запуск:** `go test -race ./twofa/internal/services/twofaService/...` (все зелёные)
- **Файловые правила:** все 4 новых файла ≤200 строк (78/83/62/172)

### Phase A bonus — ускорение тестов и Setup-flow в проде

В ходе Phase A обнаружено, что `generateBackupCodes` хеширует 10 backup-кодов **последовательно** через bcrypt cost=12 (~2.5s на каждый Setup, под `-race` ~25s × 12 setup-тестов = 270s+ по пакету).

**Что сделано:**
- `twofa/internal/services/twofaService/backup_codes.go`: bcrypt-хеширование 10 кодов вынесено в `errgroup.Group` — параллелится на всех ядрах, Setup latency падает с ~2.5s до ~250ms (×10 в проде)
- `setup_test.go::TestSetup_BackupCodeHashing`: `bcrypt.CompareHashAndPassword` × 10 тоже распараллелено через `errgroup`
- `setup_test.go`: добавлен `t.Parallel()` ко всем независимым setup-тестам (кроме `TestSetup_SecretZeroized` — мутирует package-level `GenerateSecretFunc`)

**Результат:**
- `go test ./twofa/internal/services/twofaService/...`: 5.5s (без race), 56s (с race) — было >270s
- Прод: Setup пользователя занимает ~250ms вместо ~2.5s — критично для Phase C нагрузочного тестирования

**Используемые библиотеки:**
- `golang.org/x/sync/errgroup` (уже есть в проекте) — параллельный bcrypt
- Стандартный `t.Parallel()` для независимых тестов

### Phase B — что сделано / использовано

**Path A:** mTLS как primary защита, `shared_secret` сохранён как defense-in-depth. Удалён комментарий `SECURITY(WR-03)` (закрыто mTLS).

**PKI:**
- `scripts/gen-certs.sh` — dev-only: 1 root CA (10y) + 6 leaf certs (auth, twofa, mpc-node-{1,2,3}, gateway, 825 days, EKU server+clientAuth, SANs под docker-compose имена + localhost)
- `certs/` — gitignored, `chmod 0600` на keys

**Конфигурация (per-service):**
- `TLSConfig {Enabled, CertFile, KeyFile, CAFile}` секция в `auth/twofa/mpc/gateway` config.go
- env-overrides: `<SVC>_TLS_ENABLED|CERT_FILE|KEY_FILE|CA_FILE`

**Clean Architecture файловая структура (строгое разделение):**
- `<svc>/internal/bootstrap/tls.go` — `loadServerTLSCredentials` / `loadClientTLSCredentials` (только load + validate)
- `twofa/internal/bootstrap/mpc_transport.go` / `gateway/internal/bootstrap/transport.go` — выбор TLS vs insecure (только decision logic)
- `<svc>/internal/bootstrap/<x>_clients.go` / `server.go` — wiring (использует helpers)
- `twofa/internal/adapters/mpcclient/client.go` — domain-port adapter (изолирует pb от use-case-слоя)
- `twofa/internal/middleware/client_auth.go` — client interceptor (shared_secret в metadata, defense-in-depth)

**Тесты:**
- `twofa/internal/bootstrap/tls_test.go` — `TestLoadServer/ClientTLSCredentials`, `TestMTLS_EndToEnd` (real gRPC handshake поверх loopback TCP с health-check RPC)
- Все 4 сервиса: `go build` зелёный, `go test` зелёный

**Docker:**
- `./certs:/certs:ro` volume в auth/twofa/mpc-node-{1,2,3}/gateway
- TLS env vars включены по умолчанию для production-like поведения
- **init-контейнер `certgen`** (alpine + openssl, ~10MB): запускается первым, генерирует PKI идемпотентно. Все сервисы депендят через `service_completed_successfully`. `docker compose up` без предварительных шагов поднимает работающую mTLS-mesh из чистого слейта.
- Force-regen: `rm -rf certs/ && docker compose up` или `docker compose run --rm certgen bash /work/scripts/gen-certs.sh --force`

**Параметры безопасности:**
- TLS 1.3 минимум (`MinVersion: tls.VersionTLS13`)
- `tls.RequireAndVerifyClientCert` на серверах (mTLS)
- `RootCAs` на клиентах
- gRPC автоматически проверяет SAN против dial-target

**Не сделано / отложено:**
- HMAC-подпись запросов (Path B) — отвергнуто, mTLS достаточно
- Cert rotation tooling — production concern
- CRL/OCSP — Go stdlib не поддерживает по умолчанию

### Phase C — что использовано

**Каркас:** k6 0.55.0 (Grafana) в отдельном docker-compose сервисе, общая network с приложением.

**Файловая структура (loadtest/):**
- `k6/lib/config.js` — base URL + общий пароль + thresholds
- `k6/lib/auth.js` — register/login helpers, authHeaders, uniqueEmail
- `k6/lib/totp.js` — собственная реализация TOTP в k6 (HMAC-SHA1 через `k6/crypto`, base32 decode, RFC 6238)
- `k6/login.js` — login throughput, ramping VUs 0→20
- `k6/setup-2fa.js` — setup throughput, ramping VUs 0→10
- `k6/verify-2fa.js` — verify throughput с pool 80 аккаунтов (обходит TwoFA rate limit 5/5min/user)
- `k6/mixed.js` — 70% verify / 20% login / 10% setup, реалистичный mix
- `docker-compose.loadtest.yaml` — оверлей: k6 контейнер + override gateway rate limit (иначе тест меряет лимитер)
- `loadtest/README.md` — как запускать
- `loadtest/REPORT.md` — результаты, анализ узких мест, рекомендации по масштабированию

**Makefile targets:** `load-login`, `load-setup`, `load-verify`, `load-mixed`, `load-all`.

**Ключевые цифры (Apple M3, single-host docker):**
| Endpoint | Throughput | avg latency | p95 |
|----------|------------|-------------|-----|
| Login | 22.5 RPS, 0% errors | 377 ms | 576 ms |
| Setup 2FA | 2.4 iter/s, 0% errors | 1.63 s | 2.87 s |
| Verify 2FA | 21 ms avg per call | 21.8 ms | 35.8 ms |

**Узкие места выявлены:**
- bcrypt cost=12 на login (CPU-bound, ожидаемо)
- bcrypt cost=12 на setup (10 параллельных хешей под contention)
- TwoFA rate limit 5/5min/user — корректное поведение

**Phase A улучшение валидировано под нагрузкой:** setup без parallel bcrypt был бы ~5-7s p50, измерили 1.6s.

### Phase C оптимизации (после первого load-теста)

После анализа REPORT.md прогнан пакет оптимизаций. Файлы:
- `twofa/internal/services/twofaService/backup_codes.go` — `bcryptCost = 10` (было 12). Безопасность не пострадала: коды cryptorandom (26.6 бит) + rate limit 5/5min/user
- `gateway/internal/middleware/auth_cache.go` — новый файл `TokenCache` (Redis-backed, TTL 10s, SHA-256 token hash → user_id+email)
- `gateway/internal/middleware/auth.go` — рефакторинг: extract `resolveIdentity`, accept `*TokenCache`
- `gateway/internal/bootstrap/init.go` — wire TokenCache в middleware chain
- `auth/internal/storage/pgstorage/pgstorage.go`, `twofa/.../pgstorage.go`, `mpc/.../pgstorage.go` — `MaxConns = 4×CPU`, `MinConns = CPU/2` через `runtime.NumCPU()`

**Эффект:**
- Setup p95: 2.87s → **443ms** (×6.5)
- Setup throughput: 2.4 → **5.6 iter/s** (×2.3)
- Mixed p95: 429ms → **245ms** (×1.75)
- Verify endpoint: 21ms (cache pattern в k6 не задействован — попадает в реалистичной browser-сессии)
- Login: 410ms (без изменений — bcrypt cost=12 для user passwords остался)

### Полировка по Clean Architecture / Modern Go / cc-skills-golang

**1. golangci-lint v2** (`.golangci.yml`):
- 33 линтера включены (correctness, code-quality, tests)
- Все 4 сервиса проходят с **0 issues**
- Makefile target `golangci-lint` для запуска

**2. IdentityResolver interface** (`gateway/internal/middleware/identity_resolver.go`):
- Auth middleware зависит от `IdentityResolver` interface, не от `*TokenCache`
- Реализации: `directResolver` (always RPC), `cachedResolver` (cache + fallback)
- Composition root собирает `cachedResolver(NewTokenCache(rdb), NewDirectResolver(authClient))`
- Чище для тестов — можно подменить mock'ом без gRPC

**3. Bcrypt cost через DI** (`twofa/.../twofa_service.go`):
- `Deps.BackupCodeBcryptCost` — настраиваемый параметр
- `DefaultBackupCodeBcryptCost = 10` для production
- Тесты передают `bcrypt.MinCost` (4) — тестирование пакета `twofaService` теперь **2.3s** с race / **1.6s** без race (было 56s/5.5s, изначально 270s)

**4. Бутстрап-fatalf**: вместо прямого `os.Exit(1)` после `defer cancel()` — фабрика `fatalf` с явным `cancel()` перед exit. Применено в auth, twofa, mpc init.go.

**5. Мелкая чистка:**
- `_ = conn.Close()` для best-effort cleanup
- `int64(unixTime) → unixTime` (uneeded conversion)
- `defer func() { _ = tx.Rollback(ctx) }()` (Rollback after Commit returns ErrTxDone, expected)
- gofmt sweep всего проекта

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

Система двухфакторной аутентификации на микросервисной архитектуре с распределенным хранением TOTP-секретов через протокол Shamir Secret Sharing (2-of-3). Учебный проект: Go-микросервисы, gRPC, Clean Architecture по образцу medialog/students. TOTP-секрет никогда не хранится целиком — разделяется на 3 доли, каждая хранится на отдельной MPC-ноде с шифрованием AES-256-GCM at-rest.

**Core Value:** TOTP-секрет никогда не существует в персистентном хранилище целиком — безопасность через распределение и шифрование долей.

### Constraints

- **Tech Stack**: Go 1.26.3, PostgreSQL, Redis 8.6.2, Kafka 4.1.2, gRPC, pgx (без ORM)
- **Go Modules**: `github.com/vbncursed/vkr/auth`, `github.com/vbncursed/vkr/twofa`, `github.com/vbncursed/vkr/mpc`
- **Security**: bcrypt cost=12, JWT RS256, AES-256-GCM, Shamir 2-of-3 в GF(256)
- **Architecture**: Clean Architecture (medialog/students), каждый сервис — отдельный Go-модуль
- **Academic**: все криптографические компоненты реализуются с нуля (Shamir, TOTP)
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
