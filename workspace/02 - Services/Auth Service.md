# Auth Service

## Расположение
`auth/`

## Ответственность
Регистрация, логин, выдача JWT-токенов, управление сессиями.

## gRPC API (AuthService)
| RPC | Описание |
|-----|----------|
| Register | Регистрация по email + password |
| Login | Логин, выдача access + refresh токенов |
| RefreshToken | Ротация refresh-токена |
| Logout | Удаление refresh-токена, инвалидация сессии |
| ValidateToken | Проверка access-токена, возврат user_id + claims |

## Хранение
- **PostgreSQL**: users (id, email, password_hash, created_at, updated_at), sessions, audit_log
- **Redis**: refresh-токены с TTL 7 дней

## JWT
- Алгоритм: RS256
- Access token: 15 минут
- Refresh token: 7 дней, хранится в Redis

## Валидация пароля
- Минимум 12 символов
- 1 строчная (a-z), 1 заглавная (A-Z), 1 цифра, 1 спецсимвол
- Запрет 4+ символов подряд в последовательности (1234, abcd, qwer и обратные)

## Метрики Prometheus
- `auth_requests_total{method, status}`
- `auth_request_duration_seconds`

## Kafka-события
- `user.registered`
- `user.logged_in`
- `token.refreshed`
