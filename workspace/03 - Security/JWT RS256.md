# JWT RS256

## Применение
Аутентификация пользователей. Выдается Auth Service, проверяется Gateway и другими сервисами.

## Параметры
| Параметр | Значение |
|----------|----------|
| Алгоритм | RS256 (RSA + SHA-256) |
| Access token TTL | 15 минут |
| Refresh token TTL | 7 дней |
| Хранение refresh | Redis с TTL |

## Claims (Access Token)
```json
{
  "sub": "user_id (UUID)",
  "email": "user@example.com",
  "iat": 1234567890,
  "exp": 1234568790
}
```

## Ротация Refresh Token
1. Клиент отправляет refresh_token
2. Auth проверяет наличие в Redis
3. Удаляет старый refresh_token из Redis
4. Генерирует новую пару (access + refresh)
5. Сохраняет новый refresh в Redis с TTL 7 дней

## Расположение в коде
`auth/internal/services/authService/jwt.go`
