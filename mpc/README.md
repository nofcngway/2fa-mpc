# MPC Node Service

Сервис хранения одной доли секрета Shamir. Каждая доля шифруется AES-256-GCM перед записью в PostgreSQL и дешифруется при выдаче. В production запускаются 3 независимых экземпляра с уникальными ключами шифрования.

## Стек

Go 1.26.3 · grpc v1.81.1 · pgx/v5 v5.9.2 · kafka-go v0.4.51 · prometheus/client_golang v1.23.2. AES-256-GCM — стандартные `crypto/aes` + `crypto/cipher` (без сторонних либ). Redis не используется.

## gRPC API

| RPC | Описание |
|-----|----------|
| `StoreShare` | Шифрование доли AES-256-GCM -> сохранение в PostgreSQL |
| `RetrieveShare` | Чтение из БД -> дешифрование -> возврат |
| `DeleteShare` | Удаление всех долей пользователя |

**Порты:** `9200`/`9201`/`9202` (gRPC), `9210`/`9211`/`9212` (metrics)

## Зависимости

| Компонент | Назначение |
|-----------|------------|
| PostgreSQL | Хранение зашифрованных долей |
| Kafka | События аудита (`mpc-events`) |

Redis не используется.

## Конфигурация (`config.yaml`)

```yaml
server: { port, metrics_port, log_level }
database: { dsn }
kafka: { brokers, topic }
node:
  id: 1                                        # уникальный ID ноды (1, 2, 3)
  encryption_key: "0123456789abcdef..."         # 32 байта hex для AES-256
shared_secret: "..."                            # defense-in-depth поверх mTLS
tls:
  enabled: true
  cert_file: /certs/mpc-node-1.crt              # уникальный per-node
  key_file: /certs/mpc-node-1.key
  ca_file: /certs/ca.crt
```

Каждая нода имеет свою БД и уникальный `encryption_key`.

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

- **AES-256-GCM at-rest:** каждая доля шифруется перед записью в БД, nonce генерируется через `crypto/rand`
- **mTLS:** gRPC-сервер требует client cert от TwoFA (TLS 1.3, RequireAndVerifyClientCert). Защищает от перехвата трафика и неавторизованных каллеров. См. [`docs/03 - Security/mTLS.md`](../docs/03%20-%20Security/mTLS.md)
- **Shared secret auth:** дополнительный слой defense-in-depth поверх mTLS — входящие запросы валидируются через shared secret в gRPC metadata (constant-time compare)
- **Изоляция:** каждая нода -- отдельный процесс со своей БД и ключом шифрования
- **Без внешних crypto-библиотек:** только стандартные `crypto/aes` + `crypto/cipher`

## Deployment

В docker-compose запускаются 3 экземпляра (`mpc-node-1`, `mpc-node-2`, `mpc-node-3`), каждый с:
- Уникальным `MPC_NODE_ID` (1, 2, 3)
- Уникальным `MPC_NODE_ENCRYPTION_KEY`
- Отдельной БД (`mpc_db_1`, `mpc_db_2`, `mpc_db_3`)
