# MPC-2FA

## What This Is

Двухфакторная система аутентификации с распределенным хранением TOTP-секретов через протокол Shamir Secret Sharing (2-of-3). Микросервисная архитектура на Go: Auth Service (регистрация, JWT), TwoFA Service (оркестрация 2FA, Shamir split/combine), 3 MPC-ноды (шифрованное хранение долей), API Gateway (REST→gRPC). Проект .

## Core Value

TOTP-секрет никогда не хранится целиком — он существует только транзиентно в памяти, разделяется по Shamir (2-of-3) и уничтожается. Безопасность через распределение.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] Auth Service: Register, Login, RefreshToken, Logout, ValidateToken (JWT RS256)
- [ ] TwoFA Service: Setup2FA, Verify2FA, Disable2FA, Get2FAStatus
- [ ] Shamir Secret Sharing: реализация с нуля в GF(256), 2-of-3 split/combine
- [ ] TOTP: генерация секрета, provisioning URI, валидация OTP (±1 окно)
- [ ] MPC Node Service (x3): StoreShare, RetrieveShare, DeleteShare с AES-256-GCM at-rest
- [ ] Password validation: 12+ символов, классы символов, запрет последовательностей
- [ ] Rate limiting: 5 попыток верификации 2FA за 5 минут (Redis)
- [ ] Kafka audit events для всех операций
- [ ] Prometheus метрики для всех сервисов
- [ ] gRPC Health Check Protocol в каждом сервисе
- [ ] Graceful shutdown с закрытием всех подключений
- [ ] Dockerfile, docker-compose.yaml, Makefile для каждого сервиса
- [ ] Unit-тесты: password validation, Shamir GF(256), TOTP генерация/валидация

### Out of Scope

- API Gateway — отложен на следующий этап (после backend-сервисов)
- Frontend (Next.js) — отдельный этап
- OAuth, SSO, email verification — вне ТЗ
- mTLS между сервисами — v2
- Token blacklist (emergency revocation) — v2
- MPC node backup/recovery automation — v2

## Context

- **Проект**:  (выпускная квалификационная работа)
- **Стек**: Go 1.26.2, PostgreSQL, Redis 8.6.2, Kafka 4.1.2, Prometheus, gRPC
- **Архитектура**: Clean Architecture по образцу medialog/students
- **Каждый сервис**: отдельный Go-модуль (api/, cmd/app/, config/, internal/)
- **Зависимости**: grpc, pgx/v5, go-redis/v9, kafka-go, golang-jwt/v5, prometheus, x/crypto, yaml.v3
- **Безопасность**: bcrypt cost=12, JWT RS256 (access 15мин, refresh 7д), AES-256-GCM, Shamir GF(256)
- **Порядок сборки**: Auth → TwoFA (создает proto для MPC) → MPC
- **Obsidian vault**: workspace/ — фиксировать решения, прогресс, документацию сервисов

## Constraints

- **No ORM**: pgx напрямую, без GORM
- **No HTTP in services**: только gRPC (HTTP только в Gateway)
- **No external Shamir libs**: реализация с нуля
- **No secret persistence**: TOTP-секрет только транзиентно в памяти
- **No secret logging**: пароли, доли, ключи шифрования никогда не логируются
- **Clean Architecture**: handler → service → repository, DI через bootstrap

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Backend-first (без Gateway) | Gateway зависит от всех сервисов, логичнее построить сначала backend | — Pending |
| Shamir с нуля в GF(256) | Требование  — демонстрация понимания криптографии | — Pending |
| Один бинарник для MPC-нод | Различаются только конфигурацией (NODE_ID, порт, DSN) | — Pending |
| Docker + Makefile для каждого сервиса | Запуск и сборка должны быть автоматизированы | — Pending |
| Obsidian для документации | Фиксация решений, прогресса и документации сервисов в workspace/ | — Pending |

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
*Last updated: 2026-04-11 after initialization*
