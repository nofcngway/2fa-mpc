# MPC Fault Tolerance

Описание модели отказоустойчивости системы хранения долей TOTP-секретов на 3 MPC-узлах. Документ покрывает теоретическую модель (Shamir 2-of-3) и наблюдаемое поведение каждого 2FA-флоу при частичной недоступности нод.

## Threshold-модель

Система использует Shamir Secret Sharing 2-of-3 (Shamir Secret Sharing) — TOTP-секрет разбивается на 3 доли, для восстановления нужно ровно 2.

| k | n | Восстановление возможно при | Защита от компрометации |
|---|---|------------------------------|-------------------------|
| 2 | 3 | падении 1 ноды (read-path) | компрометации 1 ноды |

Пороговое значение `k=2` выбрано как баланс: одна доля бесполезна для атакующего, но потеря одной ноды не делает аккаунт невосстановимым.

## Поведение по флоу

### Verify (read-path) — толерантен к 1 падающей ноде

Реализация: `twofa/internal/services/twofaService/retrieve_shares.go`

| Состояние нод | Поведение | Ошибка |
|---------------|-----------|--------|
| 3 живы | Verify ОК. Используются 2 первых ответивших, 3-я отменяется | — |
| 1 down + 2 живы | Verify ОК. Used 2 surviving shares | — |
| 2 down + 1 жива | Verify FAIL | `domain.ErrInsufficientShares` |
| 3 down | Verify FAIL | `domain.ErrInsufficientShares` |
| 1 slow + 2 быстрых | Verify ОК через быстрые 2 (first-2-wins). Slow node не дожидается | — |
| 1 down + 1 timeout + 1 жива | Verify FAIL после `mpc_timeout` | `domain.ErrInsufficientShares` |

**Ключевые гарантии:**
- Per-call timeout — каждый вызов RetrieveShare ограничен `cfg.mpc_timeout` (default 5s).
- Cancellation propagation — при получении 2 долей контекст для 3-й ноды отменяется (экономит ресурсы).
- No goroutine leak — даже при таймауте всех 3 нод `retrieveShares` возвращается за `mpc_timeout`, а не висит.

### Setup (write-path) — атомарность all-or-nothing

Реализация: `twofa/internal/services/twofaService/setup.go` (`distributeShares`)

| Состояние нод | Поведение | Эффект на хранилище |
|---------------|-----------|---------------------|
| 3 принимают долю | Setup ОК | `twofa_record` создан, backup-коды записаны |
| 1 нода отказывает (любая) | Setup FAIL → compensating DeleteShare на ВСЕ 3 ноды | Ничего не сохраняется |
| 2+ ноды отказывают | Setup FAIL → compensating DeleteShare | Ничего не сохраняется |

**Почему all-or-nothing:** если бы доли сохранялись частично (2 из 3), система осталась бы в неконсистентном состоянии — пользователь думает, что 2FA не подключена, а 2 ноды хранят долю и злоумышленник теоретически может восстановить секрет.

**Compensating delete использует свежий контекст** (не отменённый errgroup), чтобы успеть завершиться даже если родительский запрос отвалился.

### Disable (cleanup-path) — строгое требование 3 нод

Реализация: `twofa/internal/services/twofaService/disable.go` (`deleteSharesAll`)

| Состояние нод | Поведение | Эффект |
|---------------|-----------|--------|
| 3 принимают delete | Disable ОК | `twofa_record` удалён, backup-коды удалены, Redis cleanup |
| 1 нода отказывает | Disable FAIL | `twofa_record` НЕ удалён → 2FA остаётся активной |

**Почему строго:** оставление доли на ноде, считающей её живой, — потенциальная утечка, если позже нода будет скомпрометирована. Поэтому disable только при гарантированной очистке всех 3 нод. Пользователь видит ошибку и может ретраить.

## Тестовое покрытие

Реализовано в `twofa/internal/services/twofaService/`:

| Файл | Сценарий |
|------|----------|
| `fault_tolerance_verify_test.go` | OneNodeDown_Succeeds (×3 ноды), SlowNodeIgnored_FirstTwoWins, OneNodeTimeout_OneNodeDown_Fails, AllNodesTimeout_Fails |
| `fault_tolerance_setup_test.go` | OneNodeDown_AllOrNothing (×3 ноды) — assert compensating delete + no DB writes |
| `fault_tolerance_disable_test.go` | OneNodeDown_PreservesRecord (×3 ноды) — assert record не удалён при failure |
| `verify_test.go` | InsufficientShares (2-of-3 нод down) |
| `setup_test.go` | PartialMPCFailure_Node{0,2}Fails, AllMPCNodesFail, CompensatingDeleteUsesFreshContext |
| `disable_test.go` | ShareDeletionFails (3-я нода падает) |

Запуск: `go test -race -run TestFaultTolerance ./twofa/internal/services/twofaService/...`

## Известные ограничения

1. **Shamir restore не валидирует целостность** — если злонамеренная нода вернёт мусор вместо доли, `shamir.Combine` восстановит неверный секрет, TOTP-проверка просто не пройдёт. Защита: nodes аутентифицируются через [mTLS](mTLS.md) + shared secret, доли шифруются AES-256-GCM at-rest (см. AES-256-GCM).

2. **Setup не толерантен** к падению 1 ноды — это сознательный выбор: атомарность важнее доступности на write-path. Альтернатива (k=2 для setup тоже) допустила бы ситуации, где нода восстановится с неконсистентным состоянием.

3. **Disable требует все 3 ноды** — пользователь не сможет отключить 2FA пока хотя бы одна нода недоступна. Это компромисс в пользу безопасности: оставленная доля = потенциальная утечка.

4. **Per-call timeout = `mpc_timeout`** одинаков для всех RPC. Можно тонко настроить через config, но retry/backoff между вызовами не реализован — failure фиксируется немедленно.

## Связанные решения

- [ADR Log § ADR-001](../04%20-%20Decisions/ADR%20Log.md) — Shamir для распределённого хранения (vs. прямое хранение TOTP)
- [ADR Log § ADR-002](../04%20-%20Decisions/ADR%20Log.md) — Собственная реализация Shamir в GF(256)
- [ADR Log § ADR-003](../04%20-%20Decisions/ADR%20Log.md) — AES-256-GCM для шифрования долей at-rest

## Связанные сервисы

- TwoFA — оркестратор, реализует fault tolerance логику
- MPC Node — серверная сторона, store/retrieve/delete
- Shamir Secret Sharing
