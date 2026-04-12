# Phase 7: TwoFA Setup Flow - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-12
**Phase:** 07-twofa-setup-flow
**Areas discussed:** MPC-коммуникация, Backup-коды, Зероизация секрета, Storage и интерфейсы

---

## MPC-коммуникация

| Option | Description | Selected |
|--------|-------------|----------|
| Параллельно | 3 goroutine одновременно, errgroup с context cancel. Быстрее, но при отказе одной ноды нужно откатывать остальные | ✓ |
| Последовательно | Нода 1 → Нода 2 → Нода 3. Проще rollback, но медленнее | |
| Ты решай | Claude выберет оптимальный подход | |

**User's choice:** Параллельно (Recommended)
**Notes:** None

## Rollback Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Compensating delete | DeleteShare для всех уже сохранённых долей. DeleteShare идемпотентен (Phase 6 D-08) | ✓ |
| Оставлять orphaned shares | Не откатывать — orphans безопасны, но мусор в БД | |
| Ты решай | Claude выберет | |

**User's choice:** Compensating delete (Recommended)
**Notes:** None

---

## Backup-коды

| Option | Description | Selected |
|--------|-------------|----------|
| 8 цифр | 12345678 — просто вводить, как у Google Authenticator | |
| xxxx-xxxx (цифры с дефисом) | 1234-5678 — легче читать, но дефис нужно strip при валидации | ✓ |
| 10 алфанумерик | a1b2c3d4e5 — больше энтропии, но сложнее вводить | |
| Ты решай | Claude выберет оптимальный формат | |

**User's choice:** xxxx-xxxx (цифры с дефисом)
**Notes:** None

---

## Зероизация секрета

| Option | Description | Selected |
|--------|-------------|----------|
| defer zeroize | defer zeroize(секрет) сразу после GenerateSecret(). Гарантирует очистку на любом пути | ✓ |
| Explicit wipe | Вызывать zeroize явно в каждой точке выхода — больше контроля, но легко пропустить путь | |
| Ты решай | Claude выберет | |

**User's choice:** defer zeroize (Recommended)
**Notes:** None

---

## Storage и интерфейсы

### Email для provisioning URI

| Option | Description | Selected |
|--------|-------------|----------|
| Добавить email в прото | Добавить email в Setup2FARequest. Просто, без межсервисных запросов | ✓ |
| gRPC-запрос к Auth | TwoFA вызывает Auth.GetUser(user_id) для получения email. Создаёт зависимость TwoFA → Auth | |
| Ты решай | Claude выберет | |

**User's choice:** Добавить email в прото (Recommended)
**Notes:** None

### Повторный setup

| Option | Description | Selected |
|--------|-------------|----------|
| Запретить | Если twofa_record существует и is_enabled=true — AlreadyExists. Сначала Disable | ✓ |
| Перезаписать | Удалить старые доли и backup-коды, создать новые | |
| Ты решай | Claude выберет | |

**User's choice:** Запретить (Recommended)
**Notes:** None

---

## Claude's Discretion

- Exact errgroup pattern and error aggregation
- MPC gRPC client wrapper/pool design
- MPCClient interface vs direct gRPC stubs
- Kafka audit event structure
- Prometheus metric labels
- Internal helper decomposition

## Deferred Ideas

None — discussion stayed within phase scope
