# Сервисы и их взаимодействие

## Auth Service (`auth/`)
- Регистрация, логин, выдача JWT (RS256)
- Access-токен: 15 мин, Refresh-токен: 7 дней
- Refresh-токены хранятся в Redis с TTL
- Валидация паролей: 12+ символов, классы символов, запрет последовательностей
- Хеширование: bcrypt cost=12

## TwoFA Service (`twofa/`)
- Оркестратор 2FA
- Генерация TOTP-секрета → Shamir split → отправка долей в MPC-ноды
- Верификация: запрос 2 долей → Shamir combine → TOTP-проверка → zeroize
- Секрет НИКОГДА не персистируется
- Rate limiting: 5 попыток / 5 минут

## MPC Node (x3) (`mpc/`)
- Один бинарник, 3 инстанса (NODE_ID=1/2/3)
- Хранит одну долю секрета пользователя
- Шифрование at-rest: AES-256-GCM
- Авторизация входящих запросов через shared secret

## API Gateway (`gateway/`)
- Единая точка входа
- REST (HTTP) → gRPC трансляция
- Rate limiting (Redis)
- Маршрутизация к Auth и TwoFA сервисам

## Взаимодействие
```
Gateway ──gRPC──→ Auth Service ──→ PostgreSQL (users, sessions)
    │                    └──→ Redis (refresh tokens)
    │
    └──gRPC──→ TwoFA Service ──→ PostgreSQL (2FA metadata)
                     ├──gRPC──→ MPC Node 1 ──→ PostgreSQL (shares)
                     ├──gRPC──→ MPC Node 2 ──→ PostgreSQL (shares)
                     └──gRPC──→ MPC Node 3 ──→ PostgreSQL (shares)
```
