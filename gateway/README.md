# API Gateway

Единая точка входа для клиентских приложений. Принимает HTTP/REST-запросы и проксирует их в backend-сервисы по gRPC через grpc-gateway. Предоставляет ScalarUI для интерактивной документации API.

## REST API

| Метод | Путь | Сервис | Auth |
|-------|------|--------|------|
| POST | `/api/v1/auth/register` | Auth | -- |
| POST | `/api/v1/auth/login` | Auth | -- |
| POST | `/api/v1/auth/refresh` | Auth | -- |
| POST | `/api/v1/auth/validate` | Auth | -- |
| POST | `/api/v1/auth/logout` | Auth | + |
| POST | `/api/v1/auth/logout-all` | Auth | + |
| POST | `/api/v1/2fa/setup` | TwoFA | + |
| POST | `/api/v1/2fa/verify` | TwoFA | + |
| POST | `/api/v1/2fa/disable` | TwoFA | + |
| GET | `/api/v1/2fa/status` | TwoFA | + |

**Порт:** `8080` (HTTP), `9103` (metrics)

## Middleware

| Middleware | Описание |
|-----------|----------|
| Recovery | Перехват panic, возврат 500 |
| Metrics | Prometheus-метрики (latency, status codes) |
| Logging | Structured logging (slog) |
| CORS | Настраиваемые allowed origins |
| Rate Limiting | Ограничение запросов через Redis (requests/min + burst) |
| Auth | Валидация JWT access-токена через Auth Service |

## Зависимости

| Компонент | Назначение |
|-----------|------------|
| Redis | Rate limiting (счетчики запросов) |
| Auth Service | gRPC-подключение для проксирования + JWT-валидация |
| TwoFA Service | gRPC-подключение для проксирования |

## Конфигурация (`config.yaml`)

```yaml
server: { port, metrics_port, log_level, read_timeout, write_timeout }
auth_service: { addr: "localhost:9090" }
twofa_service: { addr: "localhost:9091" }
redis: { addr, password, db }
rate_limit: { requests_per_minute: 60, burst: 10 }
cors: { allowed_origins: ["http://localhost:3000"] }
swagger: { auth: "path/to/auth/swagger", twofa: "path/to/twofa/swagger" }
```

Переменные окружения с префиксом `GATEWAY_` переопределяют config.yaml.

## Make-команды

| Команда | Описание |
|---------|----------|
| `make generate` | Генерация protobuf + grpc-gateway кода |
| `make build` | Сборка бинарника |
| `make run` | Сборка и запуск |

## API-документация

- **ScalarUI:** http://localhost:8080/docs
- **OpenAPI specs:** генерируются из proto-файлов, хранятся в `internal/pb/swagger/`
