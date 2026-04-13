# TOTP (RFC 6238)

## Стандарт
Time-Based One-Time Password Algorithm (RFC 6238), основан на HOTP (RFC 4226).

## Параметры
| Параметр | Значение |
|----------|----------|
| Алгоритм | HMAC-SHA1 |
| Длина секрета | 20 байт (160 бит) |
| Кодировка секрета | Base32 |
| Период | 30 секунд |
| Длина кода | 6 цифр |
| Допуск | ±1 временное окно |

## Формула
```
T = floor(unix_time / 30)
HMAC = HMAC-SHA1(secret, T as 8-byte big-endian)
offset = HMAC[19] & 0x0F
code = (HMAC[offset..offset+3] & 0x7FFFFFFF) % 10^6
```

## Provisioning URI
```
otpauth://totp/MPC-2FA:{email}?secret={base32_secret}&issuer=MPC-2FA&algorithm=SHA1&digits=6&period=30
```

## Верификация
При проверке OTP-кода допускаются коды для T-1, T, T+1 (±1 окно = ±30 сек) для компенсации рассинхронизации часов.

## Расположение в коде
`twofa/internal/services/twofaService/totp/`
