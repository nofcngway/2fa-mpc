# TwoFA Service

## Расположение
`twofa/`

## Ответственность
Управление вторым фактором аутентификации. Оркестрация MPC-нод для распределенного хранения TOTP-секретов.

## gRPC API (TwoFAService)
| RPC | Описание |
|-----|----------|
| Setup2FA | Генерация секрета → Shamir split → отправка долей в MPC → provisioning URI |
| Verify2FA | Запрос 2 долей → Shamir combine → TOTP проверка |
| Disable2FA | Верификация OTP → удаление долей и метаданных |
| Get2FAStatus | Статус 2FA (is_enabled, created_at) |

## Хранение
- **PostgreSQL**: twofa_records (user_id, is_enabled, created_at), backup_codes (хеши), challenges

## Ключевые модули
### Shamir Secret Sharing (`internal/services/twofaService/shamir/`)
- Реализация с нуля в GF(256)
- Split(secret, n=3, threshold=2) → []Share
- Combine([]Share) → secret
- Интерполяция Лагранжа
- Арифметика: XOR (сложение), log/exp таблицы (умножение)

### TOTP (`internal/services/twofaService/totp/`)
- RFC 6238
- Секрет: 20 байт, base32
- Допуск: ±1 временное окно (30 сек)
- Provisioning URI: otpauth://totp/...

## Безопасность
- TOTP-секрет НИКОГДА не персистируется — zeroize после split/combine
- Rate limiting: 5 попыток верификации / 5 минут на user_id
- Таймаут gRPC к MPC-нодам: 5 секунд
- Backup-коды: 10 штук, хеши bcrypt в БД

## Метрики Prometheus
- `twofa_operations_total{operation, status}`
- `twofa_mpc_latency_seconds{node_id}`

## Kafka-события
- `2fa.enabled`
- `2fa.verified`
- `2fa.disabled`
