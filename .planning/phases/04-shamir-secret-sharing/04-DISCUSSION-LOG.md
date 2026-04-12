# Phase 4: Shamir Secret Sharing - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-12
**Phase:** 04-shamir-secret-sharing
**Areas discussed:** API дизайн Split/Combine, Тестовая стратегия, GF(256) таблицы, Расположение пакета

---

## API дизайн Split/Combine

| Option | Description | Selected |
|--------|-------------|----------|
| Package-level функции | Split(secret, n, threshold) и Combine(shares). Stateless API, нет конструктора. | ✓ |
| Методы на структуре | NewShamir(n, threshold) → .Split() и .Combine(). Фиксирует параметры при создании. | |

**User's choice:** Package-level функции
**Notes:** Чистый stateless API подходит для математического модуля без состояния.

### Follow-up: Индексация долей

| Option | Description | Selected |
|--------|-------------|----------|
| 1, 2, 3 | x=0 зарезервирован для секрета f(0), доли начинаются с 1. Мапится на MPC node ID. | ✓ |
| На усмотрение Claude | Claude выберет оптимальную схему. | |

**User's choice:** 1, 2, 3

---

## Тестовая стратегия

| Option | Description | Selected |
|--------|-------------|----------|
| Максимальный | ~25+ тестов: все комбинации 2-of-3, GF(256) арифметика, edge cases, невалидные входы. Для  нужна полнота. | ✓ |
| Базовый | ~10 тестов: roundtrip, 1-of-3 не восстанавливает, пустой/20б секрет. | |

**User's choice:** Максимальный
**Notes:**  требует демонстрации полноты тестирования криптографического модуля.

---

## GF(256) таблицы

| Option | Description | Selected |
|--------|-------------|----------|
| Runtime init() | Генерация таблиц в init() при загрузке пакета. Показывает алгоритм построения — лучше для . | ✓ |
| Hardcoded таблицы | Захардкодить 256 значений. Быстрее инициализация, но алгоритм скрыт за магическими числами. | |

**User's choice:** Runtime init()
**Notes:** Алгоритмическая генерация лучше демонстрирует понимание GF(256) на защите .

---

## Расположение пакета

| Option | Description | Selected |
|--------|-------------|----------|
| twofa/internal/crypto/shamir/ | Отдельный crypto/ пакет. Не зависит от twofaService, чистый math модуль. | ✓ |
| twofa/internal/services/twofaService/shamir/ | Как в CLAUDE.md и workspace. Вложенный пакет внутри сервиса. | |

**User's choice:** twofa/internal/crypto/shamir/
**Notes:** Крипто-код отделён от бизнес-логики. Нужно обновить CLAUDE.md и workspace doc.

---

## Claude's Discretion

- Exact helper function decomposition (polynomial evaluation, Lagrange basis)
- Error type design
- crypto/rand vs io.Reader for testability
- GF(256) function export level

## Deferred Ideas

- TOTP integration → Phase 5
- Secret zeroization → Phase 7
- Share encryption (AES-256-GCM) → Phase 6
