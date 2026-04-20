# MPC-2FA

Двухфакторная аутентификация с распределенным хранением секретов.

**-проект:** TOTP-секрет никогда не хранится целиком -- разделяется на 3 доли по протоколу Shamir Secret Sharing (2-of-3), каждая доля шифруется AES-256-GCM и хранится на отдельной MPC-ноде.

## Архитектура

```
                          ┌─────────────┐
                          │  Frontend   │
                          │  (Next.js)  │
                          └──────┬──────┘
                                 │ HTTPS / REST
                          ┌──────▼──────┐
                          │   Gateway   │
                          │  :8080      │
                          └──┬───────┬──┘
                   gRPC ┌────┘       └────┐ gRPC
                  ┌─────▼─────┐    ┌──────▼──────┐
                  │   Auth    │    │    TwoFA     │
                  │  :9090    │    │   :9091      │
                  └───────────┘    └──┬──┬──┬─────┘
                                     │  │  │  gRPC
                              ┌──────┘  │  └──────┐
                        ┌─────▼──┐ ┌────▼───┐ ┌───▼─────┐
                        │ MPC-1  │ │ MPC-2  │ │  MPC-3  │
                        │ :9200  │ │ :9201  │ │  :9202  │
                        └────────┘ └────────┘ └─────────┘
```

## Сервисы

| Сервис | Директория | Назначение |
|--------|-----------|------------|
| API Gateway | `gateway/` | REST -> gRPC, rate limiting, CORS, ScalarUI docs |
| Auth Service | `auth/` | Регистрация, логин, JWT (RS256), сессии |
| TwoFA Service | `twofa/` | Оркестрация 2FA, Shamir split/combine, TOTP |
| MPC Node (x3) | `mpc/` | Хранение одной доли секрета (AES-256-GCM at-rest) |

## Стек

| Компонент | Версия |
|-----------|--------|
| Go | 1.26.2 |
| PostgreSQL | 17 |
| Redis | 8 |
| Kafka | apache/kafka 4.1.2 |
| gRPC + gRPC-Gateway | grpc v1.80.0 |
| Prometheus | v3.4.0 |
| Grafana | 12.0.0 |

## Быстрый старт

**Предварительные требования:** Docker, Docker Compose.

```bash
# Генерация JWT-ключей для Auth Service
cd auth && make generate-keys && cd ..

# Запуск всей системы
docker compose up -d
```

| Сервис | URL |
|--------|-----|
| Gateway API | http://localhost:8080 |
| API Docs (ScalarUI) | http://localhost:8080/docs |
| Kafka UI | http://localhost:8090 |
| Prometheus | http://localhost:9190 |
| Grafana | http://localhost:3001 |

## Локальная разработка

Каждый сервис -- отдельный Go-модуль. Для работы с конкретным сервисом:

```bash
cd <service>        # auth, twofa, mpc, gateway
make generate       # сгенерировать protobuf-код
make build          # собрать бинарник
make run            # собрать и запустить
make test           # запустить тесты
```

## Структура проекта

```
.
├── auth/               # Auth Service
├── twofa/              # TwoFA Service
├── mpc/                # MPC Node Service
├── gateway/            # API Gateway
├── monitoring/         # Prometheus + Grafana конфигурация
├── scripts/            # SQL init-скрипты для Docker
├── workspace/          # Obsidian vault (документация, ADR, прогресс)
├── docker-compose.yml  # Полный dev-стек
└── CLAUDE.md           # Контекст проекта для AI
```

## Документация сервисов

- [Auth Service](auth/README.md)
- [TwoFA Service](twofa/README.md)
- [MPC Node Service](mpc/README.md)
- [API Gateway](gateway/README.md)
