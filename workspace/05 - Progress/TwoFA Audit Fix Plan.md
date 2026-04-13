# TwoFA Audit Fix Plan

**Дата**: 2026-04-13
**Источник**: [[TwoFA Service - Audit]]

## Решения по спорным пунктам

- **gRPC-сетевые**: пропускаем H-03, H-04, M-14. Фиксим H-08 (recovery interceptor) — и в twofa, и в auth
- **H-02 base32Secret**: рефакторим на `[]byte`
- **H-10 backup code verification**: реализуем сейчас
- **M-01 TOCTOU rate limit**: оставляем как есть (текущий fallback достаточен)
- **M-06 Zeroize**: `runtime.KeepAlive` после `clear()`
- **M-08/M-09 MPC abstraction**: делаем сейчас (domain types + adapter)
- **Тесты**: quality fixes + новые тесты

## Блок 1 — Critical + Security (ядро)

- [x] C-01: `NoOpSessionStorage` — fallback при недоступности Redis
- [x] H-01: `defer crypto.Zeroize(coeffs)` в `shamir.Split`
- [x] H-02: Рефакторинг `GenerateSecretFunc` на `[]byte`
- [x] M-06: `runtime.KeepAlive(b)` в `zeroize.go`
- [x] M-07: Валидация `Index != 0` в `shamir.Combine`
- [x] L-01: Zeroize share data в `retrieveShares` на error paths

## Блок 2 — Config & Startup validation

- [x] H-05: Проверка `len(mpcClients) == 3`
- [x] H-06: `Validate() error` для Config
- [x] H-07: `os.Getenv` fallback для shared_secret и DSN
- [x] H-09: GracefulStop с таймаутом
- [x] L-10/L-11: `cmp.Or` для defaults
- [x] M-11: Фикс конфликта портов в config.yaml

## Блок 3 — Recovery interceptor (twofa + auth)

- [x] H-08: Recovery interceptor в twofa
- [x] H-08: Recovery interceptor в auth

## Блок 4 — MPC abstraction (M-08/M-09)

- [x] Domain types для MPC операций
- [x] `MPCClient` interface с domain types
- [x] Adapter в bootstrap

## Блок 5 — Backup code verification (H-10)

- [x] `VerifyBackupCode` в service layer
- [x] Storage methods: `GetUnusedBackupCodeHashes`, `MarkBackupCodeUsed`
- [x] Интеграция в `Verify` (auto-detect по формату `xxxx-xxxx`)

## Блок 6 — OTP & Validation fixes

- [x] M-05: Timing-safe OTP validation (все 3 окна сравниваются всегда, `|=`)
- [x] M-13: Унифицировать OTP reuse logic (verify + disable используют одинаковый паттерн)
- [x] M-04: Sentinel error `models.ErrCounterNotFound` для "not found" в `GetUsedOTPCounter`
- [x] M-10: Переименовать метрику → `twofa_request_duration_seconds`
- [x] M-12/L-06: UUID + email validation в handlers (Setup, Verify, Disable, Status)
- [ ] L-04: Sentinel errors → domain package (частично: ErrCounterNotFound в models, остальные оставлены в service — clean arch compliant)

## Блок 7 — Тесты

- [x] Quality: обновлены mock expectations для `ErrCounterNotFound`
- [x] Новые: `verify_backup_code_test.go` (Success, InvalidCode, NoCodes, StorageError, Integration)
- [x] L-12: `bytes.Equal` в totp_test.go (уже сделано ранее)
