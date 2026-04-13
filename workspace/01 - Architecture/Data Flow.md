# Потоки данных

## Регистрация
1. User → Gateway: POST /register {email, password}
2. Gateway → Auth: gRPC Register(email, password)
3. Auth: валидация пароля → bcrypt hash → INSERT users
4. Auth → Kafka: event `user.registered`
5. Auth → Gateway → User: success

## Логин
1. User → Gateway: POST /login {email, password}
2. Gateway → Auth: gRPC Login(email, password)
3. Auth: проверка credentials → генерация JWT (access + refresh)
4. Auth → Redis: SET refresh_token (TTL 7d)
5. Auth → Kafka: event `user.logged_in`
6. Auth → Gateway → User: {access_token, refresh_token}

## Настройка 2FA (Setup)
1. User → Gateway: POST /2fa/setup (с access_token)
2. Gateway → Auth: ValidateToken → user_id
3. Gateway → TwoFA: gRPC Setup2FA(user_id)
4. TwoFA: генерация TOTP secret (20 bytes)
5. TwoFA: Shamir.Split(secret, n=3, threshold=2) → [share1, share2, share3]
6. TwoFA: **zeroize secret из памяти**
7. TwoFA → MPC Node 1: StoreShare(user_id, index=1, share1)
8. TwoFA → MPC Node 2: StoreShare(user_id, index=2, share2)
9. TwoFA → MPC Node 3: StoreShare(user_id, index=3, share3)
10. MPC Nodes: AES-256-GCM encrypt → PostgreSQL
11. TwoFA → PostgreSQL: INSERT 2fa_records (is_enabled=false)
12. TwoFA → Kafka: event `2fa.setup`
13. TwoFA → Gateway → User: provisioning URI (для QR-кода)

## Верификация 2FA
1. User → Gateway: POST /2fa/verify {otp_code}
2. Gateway → TwoFA: gRPC Verify2FA(user_id, otp_code)
3. TwoFA: rate limit check (5 попыток / 5 мин)
4. TwoFA → MPC Node 1: RetrieveShare(user_id, index=1)
5. TwoFA → MPC Node 2: RetrieveShare(user_id, index=2)
6. MPC Nodes: PostgreSQL → AES-256-GCM decrypt → return share
7. TwoFA: Shamir.Combine([share1, share2]) → secret
8. TwoFA: TOTP.Validate(secret, otp_code, ±1 window)
9. TwoFA: **zeroize secret из памяти**
10. TwoFA → PostgreSQL: UPDATE is_enabled=true (если первая верификация)
11. TwoFA → Kafka: event `2fa.verified`

## Отключение 2FA
1. User → Gateway: POST /2fa/disable {otp_code}
2. TwoFA: верификация OTP (шаги 3-9 из Verify)
3. TwoFA → MPC Nodes (x3): DeleteShare(user_id)
4. TwoFA → PostgreSQL: DELETE 2fa_records
5. TwoFA → Kafka: event `2fa.disabled`
