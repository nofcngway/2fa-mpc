# MPC Node

## Расположение
`mpc/`

## Ответственность
Хранение одной доли TOTP-секрета пользователя с шифрованием at-rest.

## Развертывание
Один бинарник, 3 инстанса. Различаются конфигурацией:
- NODE_ID (1, 2, 3)
- Порт
- PostgreSQL DSN (отдельная БД на каждую ноду)
- ENCRYPTION_KEY (уникальный для каждой ноды)

## gRPC API (MPCNodeService)
| RPC | Описание |
|-----|----------|
| StoreShare | Шифрование доли → сохранение в PostgreSQL |
| RetrieveShare | Чтение из PostgreSQL → дешифрование → возврат |
| DeleteShare | Удаление всех долей пользователя |

## Хранение
- **PostgreSQL**: shares (id UUID, user_id UUID, share_index INT, encrypted_data BYTEA, nonce BYTEA, created_at TIMESTAMP)
- UNIQUE constraint: (user_id, share_index)

## Шифрование
- AES-256-GCM
- Ключ из конфигурации (ENCRYPTION_KEY)
- Nonce: 12 байт, crypto/rand, уникальный для каждой операции
- Доли ТОЛЬКО в зашифрованном виде в БД

## Авторизация
- Shared secret через gRPC metadata ("authorization" header)
- gRPC interceptor проверяет каждый входящий запрос

## Безопасность
- share_data и encryption_key НИКОГДА не логируются
- Kafka-события содержат только user_id, share_index, operation, node_id, timestamp

## Метрики Prometheus
- `mpc_operations_total{node_id, operation, status}`
- `mpc_operation_duration_seconds`

## Kafka-события
- `share.stored`
- `share.retrieved`
- `share.deleted`
