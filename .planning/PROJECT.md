# MPC-2FA — Двухфакторная аутентификация с распределенным хранением секретов

## What This Is

Система двухфакторной аутентификации на микросервисной архитектуре с распределенным хранением TOTP-секретов через протокол Shamir Secret Sharing (2-of-3). -проект: Go-микросервисы, gRPC, Clean Architecture по образцу medialog/students. TOTP-секрет никогда не хранится целиком — разделяется на 3 доли, каждая хранится на отдельной MPC-ноде с шифрованием AES-256-GCM at-rest.

## Core Value

TOTP-секрет никогда не существует в персистентном хранилище целиком — безопасность через распределение и шифрование долей.

## Requirements

### Validated

- [x] MPC-01: StoreShare — encrypt + persist (Phase 6)
- [x] MPC-02: RetrieveShare — read + decrypt (Phase 6)
- [x] MPC-03: DeleteShare — idempotent delete (Phase 6)
- [x] MPC-04: Unique constraint (user_id, share_index) (Phase 6)
- [x] MPC-05: gRPC auth interceptor with shared secret (Phase 6)
- [x] MPC-06: AES-256-GCM encryption key from config (Phase 6)

### Active

**Auth Service (`auth/`):**
- [ ] Register — email + password, валидация пароля (12+ символов, классы, запрет последовательностей), bcrypt cost=12, Kafka-аудит
- [ ] Login — проверка credentials, JWT RS256 (access 15 мин, refresh 7 дней в Redis), Kafka-аудит
- [ ] RefreshToken — ротация refresh-токена через Redis
- [ ] Logout — удаление refresh-токена, инвалидация сессии
- [ ] ValidateToken — проверка access-токена, возврат user_id и claims

**TwoFA Service (`twofa/`):**
- [ ] Setup2FA — генерация TOTP-секрета → Shamir split (2-of-3) → отправка долей в MPC-ноды → zeroize секрета → provisioning URI + backup-коды
- [ ] Verify2FA — запрос 2 долей из MPC → Shamir combine → TOTP-валидация (±1 окно) → zeroize → rate limiting (5/5мин)
- [ ] Disable2FA — верификация + удаление долей + удаление метаданных
- [ ] Get2FAStatus — статус 2FA для пользователя
- [ ] Shamir Secret Sharing — реализация с нуля в GF(256), НЕ сторонние библиотеки
- [ ] TOTP — генерация секрета (RFC 6238), provisioning URI, валидация OTP

**MPC Node Service (`mpc/`):**
- [ ] StoreShare — шифрование AES-256-GCM + сохранение в PostgreSQL
- [ ] RetrieveShare — чтение + дешифрование
- [ ] DeleteShare — удаление всех долей пользователя
- [ ] AES-256-GCM at-rest — nonce через crypto/rand для каждой операции
- [ ] gRPC interceptor — авторизация через shared secret в metadata

**Инфраструктура (каждый сервис):**
- [ ] Clean Architecture: handler → service → repository, DI через bootstrap
- [ ] gRPC Health Check Protocol
- [ ] Graceful shutdown
- [ ] Prometheus метрики
- [ ] Structured logging (slog)
- [ ] Kafka-аудит (без секретных данных)
- [ ] Конфигурация через config.yaml

### Out of Scope

- API Gateway (`gateway/`) — отложен, не в текущем скоупе
- Frontend (Next.js) — не в текущем скоупе
- OAuth, SSO, email verification — не предусмотрено ТЗ
- ORM (GORM и т.п.) — запрещено, только pgx
- HTTP-эндпоинты в сервисах — только gRPC (HTTP только в будущем Gateway)
- Сторонние библиотеки для Shamir — реализация с нуля
- Monitoring setup (Prometheus/Grafana конфигурация) — отложено

## Context

- **** (выпускная квалификационная работа) — академический проект с фокусом на безопасности
- **Референсный проект**: medialog/students — Clean Architecture паттерн для Go-микросервисов
- **Порядок реализации**: Auth → TwoFA (создаёт proto для MPC) → MPC
- **Brownfield**: директории сервисов существуют но пусты, есть CLAUDE.md, TZ.md, workspace/ (Obsidian vault)
- **Каждый сервис** — отдельный Go-модуль (`go.mod`) на верхнем уровне
- **Proto-контракт MPC** создаётся в TwoFA и копируется в MPC

## Constraints

- **Tech Stack**: Go 1.26.2, PostgreSQL, Redis 8.6.2, Kafka 4.1.2, gRPC, pgx (без ORM)
- **Go Modules**: `github.com/vbncursed/vkr/auth`, `github.com/vbncursed/vkr/twofa`, `github.com/vbncursed/vkr/mpc`
- **Security**: bcrypt cost=12, JWT RS256, AES-256-GCM, Shamir 2-of-3 в GF(256)
- **Architecture**: Clean Architecture (medialog/students), каждый сервис — отдельный Go-модуль
- **Academic**:  — все криптографические компоненты реализуются с нуля (Shamir, TOTP)
- **Logging**: НИКОГДА не логировать секреты, пароли, доли, ключи шифрования

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Shamir 2-of-3 (не 3-of-5) | Баланс безопасности и доступности для  | — Pending |
| GF(256) для Shamir | Работает побайтово, эффективно для произвольных секретов | — Pending |
| Shared secret для MPC auth (не mTLS) | Проще для , достаточно для демонстрации | — Pending |
| RS256 для JWT (не HS256) | Асимметричная подпись — Gateway может верифицировать без секретного ключа | — Pending |
| Backup-коды хешируются bcrypt | Аналогично паролям — безопасное хранение | — Pending |
| Gateway отложен | Фокус на ядре системы: Auth + TwoFA + MPC | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-12 after Phase 6 completion*
