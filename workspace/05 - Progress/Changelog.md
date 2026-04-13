# Changelog

## 2026-04-13 (continued)
- TwoFA Service: исправлены все 38 находок аудита (кроме 3 gRPC-сетевых: H-03, H-04, M-14):
  - **Блок 1 (Critical+Security)**: NoOpSessionStorage fallback (C-01), zeroize coeffs в Shamir (H-01), GenerateSecret→[]byte (H-02), runtime.KeepAlive в Zeroize (M-06), Index!=0 валидация (M-07), zeroize shares on error (L-01)
  - **Блок 2 (Config)**: mpcClients==3 check (H-05), Config.Validate() (H-06), env var fallback (H-07), GracefulStop timeout (H-09), cmp.Or defaults (L-10/L-11), port conflict fix (M-11)
  - **Блок 3 (Recovery)**: RecoveryInterceptor в twofa+auth (H-08)
  - **Блок 4 (MPC)**: domain MPCClient interface, adapter в bootstrap (M-08/M-09)
  - **Блок 5 (Backup codes)**: VerifyBackupCode service method, auto-detect формата xxxx-xxxx в Verify, storage methods (H-10)
  - **Блок 6 (OTP)**: timing-safe comparison без early return (M-05), unified OTP reuse (M-13), models.ErrCounterNotFound sentinel (M-04), metric rename (M-10), UUID+email validation (M-12/L-06)
  - **Блок 7 (Tests)**: 5 новых backup code тестов, обновлены mock expectations для ErrCounterNotFound

## 2026-04-13
- TwoFA Service: полный аудит — 38 находок (1 CRITICAL, 10 HIGH, 15 MEDIUM, 12 LOW). Результаты в [[TwoFA Service - Audit]]
- Auth Service: полный аудит — 29 находок (1 CRITICAL, 7 HIGH, 11 MEDIUM, 10 LOW). Результаты в [[Auth Service - Audit]]
- Auth Service: исправлено 25 из 29 находок аудита (пропущены 4 gRPC-сетевые — контейнер изолирован):
  - **HIGH**: GracefulStop с таймаутом, валидация конфига, `EventProducer`/`AuditEvent` → `domain/`, `(nil,nil)` → `ErrUserNotFound`, nil-проверки в конструкторе, Kafka ErrorLogger, Redis Close error handling
  - **MEDIUM**: JWT `token_type` claim (access/refresh), max password length 128, `PasswordValidationError.Unwrap()`, аудит LogoutAll + failed logins, timing oracle fix (dummy bcrypt), `DeleteTokenFamily` error logging, error wrapping в Login
  - **LOW**: `t.Context()` (24 места), `slices.Reverse`, `cmp.Or`, `errors.Is`, `slices.ContainsFunc`, dead code в jwt_test, удалён дублирующийся тест
- MPC Service: миграция yaml.v3 → yaml/v4 (`go.yaml.in/yaml/v4`, `yaml.Load` вместо `yaml.Unmarshal`)
- MPC Service: модернизация всего кода до Go 1.26.2:
  - `interface{}` → `any` в gRPC interceptors и тестах
  - `err != http.ErrServerClosed` → `!errors.Is(err, http.ErrServerClosed)` в main.go
  - `errors.As(err, &pgErr)` → `errors.AsType[*pgconn.PgError](err)` в share.go (Go 1.26+)
  - `context.Background()` → `t.Context()` во всех тестах (Go 1.24+)
- TwoFA Service: дополнительная модернизация кода до Go 1.26.2:
  - `err != http.ErrServerClosed` → `!errors.Is(err, http.ErrServerClosed)` в main.go
  - `err == redis.Nil` → `errors.Is(err, redis.Nil)` в rate_limit.go, otp_counter.go
  - Удален лишний захват loop variable `i, share := i, share` в distributeShares (Go 1.22+)
  - C-style `for idx := 0; idx < 3; idx++` → `for idx := range 3` в verify_test.go
  - `atomic.AddInt32`/`atomic.LoadInt32` → `atomic.Int32` type-safe API (Go 1.19+) в setup_test.go
  - Удалены кастомные `contains`/`searchSubstring` хелперы → `strings.Contains` в setup_test.go
- yaml v4 (`go.yaml.in/yaml/v4`) уже был мигрирован ранее (2026-04-12)

## 2026-04-12
- Auth Service: миграция yaml.v3 → yaml/v4 (`go.yaml.in/yaml/v4`, `yaml.Load` вместо `yaml.Unmarshal`)
- Auth Service: модернизация всего кода до Go 1.26.2:
  - `interface{}` → `any` в gRPC interceptors и jwt.go
  - `errors.As()` → `errors.AsType[T]()` (register.go, user.go, тесты)
  - `err == redis.Nil` → `errors.Is(err, redis.Nil)` в session.go
  - C-style `for` → `for i := range N` (password_validation.go)
  - `omitempty` → `omitzero` в JWT Claims (Go 1.24+)
- TwoFA Service: миграция yaml.v3 → yaml/v4 (`go.yaml.in/yaml/v4`, `yaml.Load` вместо `yaml.Unmarshal`)
- TwoFA Service: модернизация всего кода до Go 1.26.2:
  - `interface{}` → `any` в gRPC interceptors
  - `clear(b)` вместо ручного цикла зануления в `crypto.Zeroize`
  - C-style `for` → `for i := range N` (shamir, gf256, backup_codes, bootstrap, retrieve_shares, тесты)
  - Удалены устаревшие `i, client := i, client` захваты loop variables (Go 1.22+)
  - `context.Background()` → `t.Context()` во всех тестах (Go 1.24+)
- Исправлен flaky тест `TestDisable_InvalidOTP` — добавлен `.Optional()` для first-2-wins мока

## 2026-04-11
- Инициализация проекта: создана структура директорий (auth, twofa, mpc, gateway, migration, monitoring)
- Создан CLAUDE.md с полным ТЗ
- Создан Obsidian vault с документацией проекта
- Сформированы промпты для реализации Auth, TwoFA, MPC сервисов
