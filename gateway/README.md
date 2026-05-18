# API Gateway

Единая точка входа для клиентских приложений. Принимает HTTP/REST-запросы и проксирует их в backend-сервисы по gRPC через grpc-gateway. Предоставляет ScalarUI для интерактивной документации API.

## Стек

Go 1.26.3 · grpc v1.81.1 · grpc-ecosystem/grpc-gateway/v2 v2.29.0 · go-redis/v9 v9.19.0 · x/sync v0.20.0 · prometheus/client_golang v1.23.2

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
tls:
  enabled: true                     # mTLS на исходящих gRPC-соединениях к Auth/TwoFA
  cert_file: /certs/gateway.crt
  key_file: /certs/gateway.key
  ca_file: /certs/ca.crt
```

Переменные окружения с префиксом `GATEWAY_` переопределяют config.yaml.

## Безопасность

- **mTLS на gRPC-клиентах:** Gateway аутентифицируется на Auth/TwoFA через client cert, валидирует server cert по project CA (TLS 1.3 минимум). См. [`docs/03 - Security/mTLS.md`](../docs/03%20-%20Security/mTLS.md)
- **JWT validation через `IdentityResolver` interface:** Auth middleware зависит от интерфейса, а не от конкретной gRPC-реализации. В composition root собирается `cachedResolver(NewTokenCache(rdb), NewDirectResolver(authClient))` — Redis-кеш validate-результатов с TTL 10s (SHA-256 hash токена → user_id+email), на miss → Auth.ValidateToken RPC. Для тестов можно подменить mock-resolver'ом
- **Rate limiting:** Redis-backed (60 req/min + burst per IP по умолчанию)
- **CORS:** explicit allowlist, никаких wildcard origin

## Make-команды

| Команда | Описание |
|---------|----------|
| `make generate` | Генерация protobuf + grpc-gateway кода |
| `make build` | Сборка бинарника |
| `make run` | Сборка и запуск |

## API-документация

- **ScalarUI:** http://localhost:8080/docs
- **OpenAPI specs:** генерируются из proto-файлов, хранятся в `internal/pb/swagger/`
