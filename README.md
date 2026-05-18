# MPC-2FA

Двухфакторная аутентификация с распределённым хранением секретов.

**-проект:** TOTP-секрет никогда не существует целиком ни в одном персистентном хранилище — он разделяется по протоколу Shamir Secret Sharing (2-of-3) и хранится тремя независимыми долями, каждая зашифрована AES-256-GCM на отдельной MPC-ноде. Внутренние gRPC-каналы защищены mutual TLS (TLS 1.3). Криптокомпоненты (Shamir, TOTP) реализованы с нуля без сторонних библиотек.

---

## Архитектура

```
                  ┌─────────────┐
                  │  Frontend   │  Next.js 16, Liquid Glass, light/dark, ru/en
                  └──────┬──────┘
                         │ HTTPS / REST
                  ┌──────▼──────┐
                  │   Gateway   │  REST → gRPC, JWT, rate limit, ScalarUI
                  │  :8080      │
                  └──┬───────┬──┘
              mTLS   │       │   mTLS
              ┌──────▼┐   ┌──▼──────┐
              │ Auth  │   │ TwoFA   │  Shamir 2-of-3, TOTP RFC 6238
              │ :9090 │   │ :9091   │
              └───────┘   └─┬───┬─┬─┘
                  mTLS ┌────┘   │ └────┐  mTLS
              ┌────────▼┐  ┌────▼──┐ ┌─▼────────┐
              │ MPC-1   │  │ MPC-2 │ │ MPC-3    │  AES-256-GCM at-rest,
              │ :9200   │  │ :9201 │ │ :9202    │  per-node encryption key
              └─────────┘  └───────┘ └──────────┘
```

Все межсервисные каналы — mTLS (см. [`docs/03 - Security/mTLS.md`](docs/03%20-%20Security/mTLS.md)). Frontend → Gateway — обычный HTTPS (терминируется внешним прокси в проде).

---

## Сервисы

| Сервис | Директория | Назначение |
|--------|-----------|------------|
| Frontend | `frontend/` | Next.js 16, HeroUI, Liquid Glass design system |
| API Gateway | `gateway/` | REST → gRPC, JWT-валидация, rate limiting, CORS, ScalarUI docs |
| Auth Service | `auth/` | Регистрация, логин, JWT (RS256), refresh-сессии, audit |
| TwoFA Service | `twofa/` | Оркестрация 2FA, Shamir split/combine, TOTP, backup-коды, rate limiting |
| MPC Node (×3) | `mpc/` | Хранение одной доли секрета (AES-256-GCM at-rest) |

---

## Стек

| Компонент | Версия |
|-----------|--------|
| Go | 1.26.3 |
| PostgreSQL | 17 |
| Redis | 8 |
| Kafka | apache/kafka 4.1.2 |
| gRPC + gRPC-Gateway | grpc v1.80.0 |
| Prometheus | v3.4.0 |
| Grafana | 12.0.0 |
| Frontend | Next.js 16, React 19, Tailwind v4, HeroUI |

---

## Безопасность

| Слой | Механизм | Реализация |
|------|----------|-------------|
| Транспорт между сервисами | mTLS (TLS 1.3, RequireAndVerifyClientCert) | `<svc>/internal/bootstrap/tls.go` |
| Identity сервисов | per-service x509 cert + единый CA | `scripts/gen-certs.sh`, авто-генерация в docker-compose |
| Defense-in-depth | shared_secret в gRPC metadata | `<svc>/internal/middleware/client_auth.go` |
| Хранение TOTP | Shamir Secret Sharing 2-of-3 в GF(256) | `twofa/internal/crypto/shamir/` (with-scratch) |
| Шифрование долей | AES-256-GCM, nonce per-share | `mpc/internal/services/mpcService/encrypt.go` |
| TOTP | RFC 6238 ±1 window, OTP reuse prevention | `twofa/internal/crypto/totp/` (with-scratch) |
| JWT | RS256, access 15м / refresh 7д, token families | `auth/internal/services/auth_service/jwt.go` |
| Пароли | bcrypt cost=12 + строгая валидация | `auth/internal/services/auth_service/password_validation.go` |
| Backup-коды | 10 шт. xxxx-xxxx, bcrypt-hash, parallel generation | `twofa/internal/services/twofaService/backup_codes.go` |
| Fault tolerance | 2-of-3 — Verify работает при падении 1 ноды | `twofa/internal/services/twofaService/fault_tolerance_*_test.go` |
| Rate limiting | Gateway: 60 req/min per IP; TwoFA: 5 verify/5min per user | Redis-backed |

Подробнее: [`docs/03 - Security/`](docs/03%20-%20Security/), [`docs/04 - Decisions/ADR Log.md`](docs/04%20-%20Decisions/ADR%20Log.md).

---

## Быстрый старт

**Требования:** Docker, Docker Compose.

```bash
# 1. Сгенерировать JWT-ключи для Auth Service (один раз)
cd auth && make generate-keys && cd ..

# 2. Запустить всю систему
make up-build
```

mTLS-сертификаты генерируются автоматически init-контейнером `certgen` при первом `docker compose up` — отдельных шагов делать не нужно. PKI кладётся в `./certs/` (gitignored).

| Сервис | URL |
|--------|-----|
| Frontend | http://localhost:3000 |
| Gateway API | http://localhost:8080 |
| API Docs (ScalarUI) | http://localhost:8080/docs |
| Kafka UI | http://localhost:8090 |
| Prometheus | http://localhost:9190 |
| Grafana | http://localhost:3001 |

Принудительная регенерация сертификатов:

```bash
rm -rf certs/ && docker compose up
# или
docker compose run --rm certgen bash /work/scripts/gen-certs.sh --force
```

---

## Локальная разработка

Каждый сервис — отдельный Go-модуль:

```bash
cd <service>        # auth, twofa, mpc, gateway
make generate       # protobuf-код
make build          # бинарник
make run            # собрать и запустить
make test           # тесты
```

Тесты включают:
- Unit-тесты криптокомпонентов (Shamir GF(256), TOTP)
- Service-layer тесты с minimock
- **Fault tolerance suite** для 2-of-3 threshold модели (`fault_tolerance_*_test.go`)
- mTLS handshake integration test (`tls_test.go`)

Запуск всех тестов из корня:

```bash
make test-all          # go test ./... во всех 4 сервисах
make lint-all          # go vet
make golangci-lint     # golangci-lint v2 (33 линтера) — все сервисы 0 issues
```

Тесты быстрые благодаря `Deps.BackupCodeBcryptCost = bcrypt.MinCost` в test-suites — пакет `twofaService` тестируется за **2.3s** (с race) / **1.6s** (без race).

## Нагрузочное тестирование

```bash
make up-build         # поднять стек
make load-all         # прогнать все 4 сценария k6 (login, setup, verify, mixed)
```

Подробности и результаты — в [`loadtest/REPORT.md`](loadtest/REPORT.md).

---

## Структура проекта

```
.
├── frontend/           # Next.js 16, HeroUI, Liquid Glass
├── auth/               # Auth Service
├── twofa/              # TwoFA Service
├── mpc/                # MPC Node Service
├── gateway/            # API Gateway
├── monitoring/         # Prometheus + Grafana конфигурация
├── scripts/
│   ├── gen-certs.sh    # PKI generator (CA + 6 leaf certs, TLS 1.3)
│   └── init-*.sql      # PostgreSQL init
├── certs/              # gitignored — сгенерированные mTLS-сертификаты
├── docs/               # ADR, security deep-dives, changelog
├── docker-compose.yml  # Полный dev-стек с автогенерацией PKI
├── README.md           # этот файл
└── CLAUDE.md           # AI-context: текущий статус, правила, Phase log
```

---

##  — статус закрытия пунктов ТЗ

| Phase | Описание | Owner | Status |
|-------|----------|-------|--------|
| A | MPC fault tolerance — тесты 2-of-3 threshold + документация |  | ✅ Done |
| B | mTLS между всеми сервисами + автогенерация PKI + ADR-011 |  | ✅ Done |
| C | k6 нагрузочные тесты + REPORT.md + рекомендации по масштабированию |  | ✅ Done |
| D | Frontend monitoring page (Prometheus widgets) |  | ✅ Done |

Полный лог изменений: [`docs/05 - Progress/Changelog.md`](docs/05%20-%20Progress/Changelog.md).

---

## Документация сервисов

- [Frontend](frontend/README.md)
- [Auth Service](auth/README.md)
- [TwoFA Service](twofa/README.md)
- [MPC Node Service](mpc/README.md)
- [API Gateway](gateway/README.md)

## Документация проекта

Глубокая документация — в [`docs/`](docs/README.md):

| Раздел | Что внутри |
|--------|-----------|
| [`docs/03 - Security/`](docs/03%20-%20Security/) | mTLS, MPC Fault Tolerance — security deep-dives |
| [`docs/04 - Decisions/ADR Log.md`](docs/04%20-%20Decisions/ADR%20Log.md) | ADR-001..011 — архитектурные решения с обоснованием |
| [`docs/05 - Progress/Changelog.md`](docs/05%20-%20Progress/Changelog.md) | Лог всех значимых изменений по фазам  |

---

## Лицензия

[LICENSE](LICENSE)
