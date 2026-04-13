# AES-256-GCM шифрование

## Применение
Шифрование долей (shares) at-rest в MPC-нодах.

## Параметры
| Параметр | Значение |
|----------|----------|
| Алгоритм | AES-256-GCM |
| Длина ключа | 32 байта (256 бит) |
| Длина nonce | 12 байт (96 бит) |
| Источник nonce | crypto/rand |

## Схема
```
Encrypt(key, plaintext):
  nonce = crypto/rand (12 bytes)
  ciphertext = AES-256-GCM.Seal(nonce, key, plaintext, nil)
  return (ciphertext, nonce)

Decrypt(key, ciphertext, nonce):
  plaintext = AES-256-GCM.Open(nonce, key, ciphertext, nil)
  return plaintext
```

## Хранение в PostgreSQL
```sql
shares (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL,
  share_index INT NOT NULL,
  encrypted_data BYTEA NOT NULL,  -- зашифрованная доля
  nonce BYTEA NOT NULL,           -- уникальный nonce
  created_at TIMESTAMP NOT NULL,
  UNIQUE(user_id, share_index)
)
```

## Ключ шифрования
- Уникальный для каждой MPC-ноды
- Загружается из конфигурации (ENCRYPTION_KEY)
- НИКОГДА не логируется
- НИКОГДА не передается по сети

## Расположение в коде
`mpc/internal/crypto/aes.go`
