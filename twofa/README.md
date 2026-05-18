# TwoFA Service

Сервис оркестрации двухфакторной аутентификации. Генерирует TOTP-секрет, разделяет его на 3 доли по протоколу Shamir Secret Sharing (2-of-3), распределяет доли по MPC-нодам. При верификации собирает 2 доли, восстанавливает секрет, проверяет OTP-код и немедленно зануляет секрет в памяти.

## Стек

Go 1.26.3 · grpc v1.81.1 · pgx/v5 v5.9.2 · go-redis/v9 v9.19.0 · kafka-go v0.4.51 · x/sync (errgroup) v0.20.0 · prometheus/client_golang v1.23.2. Криптокомпоненты (Shamir GF(256), TOTP RFC 6238) — собственная реализация без сторонних либ.

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
shared_secret: "..."           # defense-in-depth поверх mTLS
mpc_timeout: 5s
tls:
  enabled: true                # mTLS на server и client (MPC) сторонах
  cert_file: /certs/twofa.crt
  key_file: /certs/twofa.key
  ca_file: /certs/ca.crt
```

Env overrides: `TWOFA_SHARED_SECRET`, `TWOFA_DATABASE_DSN`, `TWOFA_MPC_NODES`, `TWOFA_TLS_*`.

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
- **Backup-коды:** 10 штук cryptorandom 8-digit, bcrypt-хеши (cost=10 — настраивается через `Deps.BackupCodeBcryptCost`), **параллельная генерация через errgroup**. Setup p95 под 10 VU: 2.87s → **443ms**. Ниже cost user-password (12) обоснованно: 26.6 бит энтропии + rate limit 5/5min/user — brute-force нереалистичен
- **Параллельные вызовы MPC:** через `errgroup` для минимизации latency
- **Fault tolerance:** Verify работает при падении 1 ноды (Shamir 2-of-3); Setup и Disable атомарны (all-or-nothing). Тесты: `fault_tolerance_*_test.go`. См. [`docs/03 - Security/MPC Fault Tolerance.md`](../docs/03%20-%20Security/MPC%20Fault%20Tolerance.md)
- **mTLS:** gRPC-сервер требует client cert от Gateway; gRPC-клиенты к MPC аутентифицируются client cert. TLS 1.3 минимум. См. [`docs/03 - Security/mTLS.md`](../docs/03%20-%20Security/mTLS.md)
