# Auth Service

Сервис аутентификации и управления сессиями. Регистрация пользователей, логин, выдача и ротация JWT-токенов (RS256), управление сессиями через token families.

## Стек

Go 1.26.3 · grpc v1.81.1 · pgx/v5 v5.9.2 · go-redis/v9 v9.19.0 · kafka-go v0.4.51 · golang-jwt/jwt/v5 v5.3.1 · prometheus/client_golang v1.23.2

## gRPC API

| RPC | Описание |
|-----|----------|
| `Register` | Регистрация по email + password |
| `Login` | Логин, выдача access + refresh токенов |
| `RefreshToken` | Ротация refresh-токена (token family) |
| `Logout` | Инвалидация refresh-токена |
| `LogoutAll` | Удаление всех сессий пользователя (Lua-скрипт) |
| `ValidateToken` | Проверка access-токена, возврат user_id + claims |

**Порт:** `9090` (gRPC), `9100` (metrics)

## Зависимости

| Компонент | Назначение |
|-----------|------------|
| PostgreSQL | Хранение пользователей (`users`) |
| Redis | Refresh-токены, token families, сессии (TTL) |
| Kafka | События аудита (`auth-events`) |

## Конфигурация (`config.yaml`)

```yaml
server: { port, metrics_port, log_level }
database: { dsn }
redis: { addr, password, db }
kafka: { brokers, topic }
jwt:
  private_key_path: keys/private.pem
  public_key_path: keys/public.pem
  access_token_ttl: 15m
  refresh_token_ttl: 168h   # 7 дней
tls:
  enabled: true
  cert_file: /certs/auth.crt
  key_file: /certs/auth.key
  ca_file: /certs/ca.crt
```

Переменные окружения с префиксом `AUTH_` переопределяют config.yaml.

## Make-команды

| Команда | Описание |
|---------|----------|
| `make generate` | Генерация protobuf-кода |
| `make mock` | Генерация моков (minimock) |
| `make build` | Сборка бинарника |
| `make run` | Сборка и запуск |
| `make test` | Запуск тестов |
| `make generate-keys` | Генерация RSA-ключей для JWT |
| `make lint` | go vet + golangci-lint |

## Безопасность

- **Пароли:** bcrypt cost=12, валидация сложности (12+ символов, uppercase, lowercase, цифра, спецсимвол, запрет последовательностей)
- **JWT:** RS256 (асимметричная подпись), access 15 мин, refresh 7 дней
- **Token families:** обнаружение повторного использования refresh-токенов -- при детекции инвалидируется вся семья
- **Timing-safe:** логин возвращает одинаковую ошибку для несуществующего email и неверного пароля
- **mTLS:** gRPC-сервер требует client cert (TLS 1.3, RequireAndVerifyClientCert). Только Gateway с валидным сертификатом может вызвать internal RPC. См. [`docs/03 - Security/mTLS.md`](../docs/03%20-%20Security/mTLS.md)
