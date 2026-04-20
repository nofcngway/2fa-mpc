# TwoFA Service

Сервис оркестрации двухфакторной аутентификации. Генерирует TOTP-секрет, разделяет его на 3 доли по протоколу Shamir Secret Sharing (2-of-3), распределяет доли по MPC-нодам. При верификации собирает 2 доли, восстанавливает секрет, проверяет OTP-код и немедленно зануляет секрет в памяти.

## gRPC API

| RPC | Описание |
|-----|----------|
| `Setup2FA` | Генерация секрета -> Shamir split -> отправка долей на MPC-ноды -> provisioning URI + backup-коды |
| `Verify2FA` | Запрос 2 долей -> Shamir combine -> TOTP-проверка -> zeroize |
| `Disable2FA` | Верификация OTP -> удаление долей и метаданных |
| `Get2FAStatus` | Статус 2FA (is_enabled, created_at) |

**Порт:** `9091` (gRPC), `9101` (metrics)

## Зависимости

| Компонент | Назначение |
|-----------|------------|
| PostgreSQL | 2FA-записи, хеши backup-кодов |
| Redis | Rate limiting (5 попыток / 5 мин), OTP-счетчики |
| MPC-ноды (x3) | Хранение долей секрета (gRPC) |
| Kafka | События аудита (`twofa-events`) |

## Конфигурация (`config.yaml`)

```yaml
server: { port, metrics_port, log_level }
database: { dsn }
redis: { addr, password, db }
kafka: { brokers, topic }
mpc_nodes: ["node1:9200", "node2:9201", "node3:9202"]
shared_secret: "..."           # авторизация на MPC-нодах
mpc_timeout: 5s
```

Env overrides: `TWOFA_SHARED_SECRET`, `TWOFA_DATABASE_DSN`, `TWOFA_MPC_NODES`.

## Make-команды

| Команда | Описание |
|---------|----------|
| `make generate` | Генерация protobuf-кода |
| `make mock` | Генерация моков (minimock) |
| `make build` | Сборка бинарника |
| `make run` | Сборка и запуск |
| `make test` | Запуск тестов |
| `make lint` | go vet + golangci-lint |

## Безопасность

- **TOTP-секрет** никогда не персистируется целиком -- существует только транзиентно в памяти
- **Zeroization:** секрет занулируется в памяти сразу после использования (`crypto/zeroize.go`)
- **Shamir SSS:** собственная реализация в GF(256), порог 2-of-3 (`internal/crypto/shamir/`)
- **TOTP:** собственная реализация RFC 6238, окно +/-1 (`internal/crypto/totp/`)
- **Rate limiting:** максимум 5 попыток верификации за 5 минут на user_id (Redis)
- **Backup-коды:** 10 штук, хранятся как bcrypt-хеши, constant-time сравнение
- **Параллельные вызовы MPC:** через `errgroup` для минимизации latency
