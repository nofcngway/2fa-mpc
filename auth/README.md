# Auth Service

Сервис аутентификации и управления сессиями. Регистрация пользователей, логин, выдача и ротация JWT-токенов (RS256), управление сессиями через token families.

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
