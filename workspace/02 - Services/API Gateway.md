# API Gateway

## Расположение
`gateway/`

## Ответственность
Единая точка входа. Трансляция REST (HTTP) → gRPC. Rate limiting.

## Протоколы
- **Вход**: HTTP/REST от Frontend
- **Выход**: gRPC к Auth Service и TwoFA Service

## Маршруты (HTTP → gRPC)
| HTTP | gRPC Target |
|------|-------------|
| POST /register | Auth.Register |
| POST /login | Auth.Login |
| POST /refresh | Auth.RefreshToken |
| POST /logout | Auth.Logout |
| POST /2fa/setup | TwoFA.Setup2FA |
| POST /2fa/verify | TwoFA.Verify2FA |
| POST /2fa/disable | TwoFA.Disable2FA |
| GET /2fa/status | TwoFA.Get2FAStatus |

## Rate Limiting
- Счетчики в Redis
- Ограничения по IP и по user_id

## Middleware
- JWT-валидация (через Auth.ValidateToken) для защищенных эндпоинтов
- Rate limiting
- CORS
- Request logging
