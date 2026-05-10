# ADR Log

## ADR-001 — Shamir Secret Sharing для распределённого хранения секретов
- **Дата**: 2026-04
- **Статус**: Accepted
- **Контекст**: TOTP-секрет нельзя хранить целиком ни в одной БД (тогда компрометация одного хранилища = компрометация секретов всех пользователей).
- **Решение**: Shamir Secret Sharing 2-of-3 — 3 доли на 3 нодах, для восстановления нужно 2.
- **Причина**: Криптографически доказуемая защита от компрометации 1 ноды. Терпимость к падению 1 ноды.
- **Связано**: TwoFA, MPC Node, Shamir Secret Sharing, [MPC Fault Tolerance](../03%20-%20Security/MPC%20Fault%20Tolerance.md)

## ADR-002 — Собственная реализация Shamir в GF(256)
- **Дата**: 2026-04
- **Статус**: Accepted
- **Контекст**:  требует реализации криптокомпонентов с нуля.
- **Решение**: реализация Shamir в `twofa/internal/crypto/shamir/` через GF(256).
- **Связано**: TwoFA

## ADR-003 — AES-256-GCM для шифрования долей at-rest
- **Дата**: 2026-04
- **Статус**: Accepted
- **Контекст**: Доли в БД ноды должны быть зашифрованы — компрометация БД не должна давать доступ к долям без ключа ноды.
- **Решение**: AES-256-GCM, nonce через crypto/rand, ключ в `MPC_NODE_ENCRYPTION_KEY`.
- **Связано**: MPC Node

## ADR-004 — RS256 JWT (vs HS256)
- **Дата**: 2026-04
- **Статус**: Accepted
- **Контекст**: Нужна асимметричная подпись для децентрализованной верификации (Gateway проверяет access-токен без секрета).
- **Решение**: RS256, приватный ключ только в Auth, публичный — везде.
- **Связано**: Auth, JWT RS256

## ADR-011 — mTLS между всеми внутренними сервисами
- **Дата**: 2026-05-10
- **Статус**: Accepted
- **Контекст**: До Phase B всё внутреннее общение (Gateway↔Auth/TwoFA, TwoFA↔MPC) шло через `insecure.NewCredentials()`. Идентификация сервисов опиралась только на shared_secret в gRPC metadata, что не защищает от перехвата трафика и не даёт криптографической идентификации каллера. Старый комментарий `SECURITY(WR-03)` явно отмечал это как незакрытый долг.
- **Решение**: Включить mTLS на всех internal gRPC-каналах с единым CA, выпустить per-service сертификаты с SAN под docker-compose имена. shared_secret сохранён как defense-in-depth. Cert generation через `scripts/gen-certs.sh`.
- **Причина**:
  1. mTLS даёт криптографическую идентификацию обеих сторон (server + client cert) — закрывает WR-03.
  2. TLS 1.3 шифрует транспорт — защищает от перехвата трафика в внутренней сети.
  3. shared_secret (на уровне metadata) сохранён как defense-in-depth: если кто-то отключит TLS по ошибке, downstream-сервер всё равно отвергнёт unauth-запрос.
  4. Не выбран HMAC-подпись запросов: TLS уже защищает от replay внутри сессии (sequence numbers), а cross-session replay требует подделку TLS handshake.
- **Последствия**:
  - Все 4 сервиса конфигурируют TLS (cert + key + ca) через config.
  - Operator должен сгенерировать сертификаты до запуска (`scripts/gen-certs.sh`).
  - Local dev без TLS остаётся возможен (`*_TLS_ENABLED=false`) с loud warning.
  - Production деплой обязан включать TLS (нет fail-fast — но warning).
  - Удалён `SECURITY(WR-03)` комментарий из `auth/internal/api/auth_service_api/logout_all.go`.
- **Связано**: Auth, TwoFA, MPC Node, Gateway, [mTLS](../03%20-%20Security/mTLS.md)
