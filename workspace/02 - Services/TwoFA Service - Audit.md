# TwoFA Service — Аудит

**Дата**: 2026-04-13
**Версия Go**: 1.26.2
**Аудит**: cc-skills-golang + golang-mastery-skill (5 параллельных агентов)

## Сводка

| Severity | Количество |
|----------|-----------|
| CRITICAL | 1 |
| HIGH     | 10 |
| MEDIUM   | 15 |
| LOW      | 12 |
| **Итого** | **38** |

---

## CRITICAL

### C-01. Nil SessionStorage → panic при недоступности Redis
- **Файл**: `bootstrap/bootstrap.go:30-37`, `cmd/app/main.go:49`
- `NewRedisStorage` возвращает `nil` при ошибке Redis. `Verify()` и `Disable()` вызывают методы `sessionStorage` напрямую → nil pointer panic. Сервис падает целиком.
- **Fix**: Создать `NoOpSessionStorage` (аналог `NoOpProducer` для Kafka), возвращать его вместо `nil`.

---

## HIGH

### H-01. Coefficients buffer не зeroize после Split
- **Файл**: `crypto/shamir/shamir.go:89`
- Слайс `coeffs` содержит байт секрета (`coeffs[0]`) и случайные коэффициенты. После возврата из `Split` буфер остается в памяти. Вызывающий код не имеет к нему доступа.
- **Fix**: `defer crypto.Zeroize(coeffs)` после `coeffs := make([]byte, threshold)`.

### H-02. base32Secret (string) невозможно зeroize в Setup
- **Файл**: `services/twofaService/setup.go:48`
- `GenerateSecretFunc()` возвращает `base32Secret string`. Go strings immutable — строка с полным TOTP-секретом в base32 живет до GC. Передается далее в `GenerateProvisioningURI`.
- **Fix**: Работать с `[]byte` вместо `string` для base32-секрета, конвертировать в string только при gRPC-сериализации, зeroize буфер. Или задокументировать как ограничение Go.

### H-03. Insecure gRPC transport к MPC-нодам
- **Файл**: `bootstrap/bootstrap.go:48`
- `grpc.WithTransportCredentials(insecure.NewCredentials())` — доли передаются в plaintext по сети.
- **Fix**: TLS/mTLS для MPC-соединений. (gRPC-сетевое — **пропускаем**, контейнер изолирован)

### H-04. Shared secret в plaintext metadata
- **Файл**: `bootstrap/bootstrap.go:65-77`
- `SharedSecret` передается в `authorization` gRPC metadata без шифрования транспорта.
- **Fix**: mTLS вместо статического bearer token. (gRPC-сетевое — **пропускаем**)

### H-05. Нет валидации количества MPC-нод
- **Файл**: `bootstrap/bootstrap.go:42`, `cmd/app/main.go:51`
- Shamir 2-of-3 требует ровно 3 ноды. Нет проверки `len(mpcClients) == 3`. При меньшем количестве → panic index out of range.
- **Fix**: Проверка при старте: `if len(cfg.MPCNodes) != 3 { log.Fatal(...) }`.

### H-06. Нет валидации конфига
- **Файл**: `config/config.go:64-76`
- `Load()` только парсит YAML. Нет проверки: пустой DSN, нулевой порт, пустой shared secret, пустые адреса MPC-нод, отсутствие Kafka brokers.
- **Fix**: Метод `Validate() error` с `errors.Join`, вызов из `Load()`.

### H-07. Shared secret в plaintext config.yaml
- **Файл**: `config.yaml:24`
- `shared_secret: "dev-shared-secret-change-in-production"` — нет механизма чтения из env variables.
- **Fix**: Поддержка `os.Getenv` fallback для `shared_secret`.

### H-08. Нет recovery interceptor
- **Файл**: `bootstrap/bootstrap.go:104-108`
- gRPC server имеет `MetricsInterceptor` + `LoggingInterceptor`, но нет recovery. Panic в хендлере (напр. от C-01) → crash всего процесса.
- **Fix**: Recovery interceptor первым в цепочке. (gRPC-сетевое — **пропускаем**)

### H-09. GracefulStop без таймаута
- **Файл**: `cmd/app/main.go`
- `grpcServer.GracefulStop()` блокирует бесконечно при зависших соединениях.
- **Fix**: Горутина + select с таймаутом, fallback на `Stop()`.

### H-10. Backup code verification не реализован
- **Файл**: `services/twofaService/` — отсутствует логика верификации backup-кодов
- `Verify` метод валидирует только TOTP-коды. Backup-коды генерируются и сохраняются, но никогда не проверяются. Пользователи не могут восстановить доступ через backup-коды.
- **Fix**: Реализовать верификацию: bcrypt compare, one-time-use (удаление после использования), rate limiting.

---

## MEDIUM

### M-01. TOCTOU race в rate limiting (INCR + EXPIRE)
- **Файл**: `storage/redisstorage/rate_limit.go:13-26`
- `INCR` и `EXPIRE` — две отдельные Redis-команды. При crash между ними ключ живет без TTL → перманентная блокировка пользователя.
- **Fix**: Lua-скрипт: `local c = redis.call('INCR', KEYS[1]); if c == 1 then redis.call('EXPIRE', KEYS[1], ARGV[1]) end; return c`.

### M-02. Rate limiting bypass при падении Redis
- **Файл**: `services/twofaService/verify.go:58-60`
- При ошибке Redis код логирует warning и продолжает — rate limiting отключается. Атакующий может brute-force OTP при недоступном Redis.
- **Fix**: In-memory fallback rate limiter или задокументировать risk.

### M-03. OTP reuse check пропускается при падении Redis
- **Файл**: `services/twofaService/verify.go:89-94`
- При ошибке `GetUsedOTPCounter` → `hasLastCounter = false` → один OTP можно использовать многократно в 90-секундном окне.
- **Fix**: Тот же подход, что и M-02.

### M-04. GetUsedOTPCounter возвращает 0 для "not found"
- **Файл**: `storage/redisstorage/otp_counter.go:21`
- `redis.Nil` → `(0, nil)`. В verify.go `hasLastCounter = true`, `lastCounter = 0`. Теоретическая неоднозначность (counter=0 unreachable на практике).
- **Fix**: Вернуть sentinel error или wrapper type.

### M-05. Early-return timing leak в OTP validation
- **Файл**: `crypto/totp/totp.go:56-67`
- `subtle.ConstantTimeCompare` для каждого окна, но return `true` на первом совпадении. Атакующий может измерить, какое окно совпало.
- **Fix**: Проверить все 3 окна, объединить результат в конце.

### M-06. Zeroize через `clear()` может быть оптимизирован компилятором
- **Файл**: `crypto/zeroize.go:6`
- Go `clear()` может быть удален dead store elimination, если компилятор определит, что слайс не читается после.
- **Fix**: Explicit loop + `runtime.KeepAlive(b)`, или function variable для предотвращения inlining.

### M-07. Нет валидации share Index != 0 в Combine
- **Файл**: `crypto/shamir/shamir.go:111-157`
- `Combine` проверяет дубликаты и пустые данные, но не отклоняет `Index == 0`. x=0 — это сам секрет. Доля с Index=0 даст неверный результат без ошибки.
- **Fix**: `if s.Index == 0 { return nil, ErrInvalidShareIndex }`.

### M-08. Service imports protobuf types из mpc_api
- **Файл**: `services/twofaService/setup.go:14`, `retrieve_shares.go:9`, `disable.go:14`, `twofa_service.go:9`
- Бизнес-логика импортирует `mpc_api` protobuf типы напрямую. Service layer связан с gRPC transport внешнего сервиса.
- **Fix**: Domain types для MPC + adapter в bootstrap.

### M-09. MPCClient interface зеркалит generated gRPC client
- **Файл**: `services/twofaService/twofa_service.go:38-42`
- `MPCClient` interface содержит `grpc.CallOption` → привязка domain layer к gRPC.
- **Fix**: Более простой domain interface без gRPC-specifics.

### M-10. Метрика `twofa_mpc_latency_seconds` — misleading name
- **Файл**: `middleware/metrics.go:17`
- Гистограмма названа `twofa_mpc_latency_seconds`, но измеряет latency ВСЕХ gRPC handler'ов, не только MPC.
- **Fix**: Переименовать в `twofa_request_duration_seconds`.

### M-11. MPC node port конфликтует с metrics port
- **Файл**: `config.yaml:5,21`
- `metrics_port: 9101` и `mpc_nodes[1].addr: "localhost:9101"` — один и тот же порт.
- **Fix**: Назначить неперекрывающиеся порты.

### M-12. UUID формат user_id не валидируется в Setup handler
- **Файл**: `api/twofa_service_api/setup.go:16`
- Проверка только `req.UserId == ""`, без UUID-формата. Invalid string → PostgreSQL error → `codes.Internal` вместо `codes.InvalidArgument`.
- **Fix**: UUID format validation в handler.

### M-13. Inconsistent OTP reuse logic между Verify и Disable
- **Файл**: `services/twofaService/disable.go:59`
- Disable добавляет `&& matchedCounter != 0`, Verify — нет.
- **Fix**: Унифицировать паттерн проверки OTP reuse.

### M-14. Нет server-side auth interceptor
- **Файл**: `middleware/interceptors.go`
- TwoFA сервис не проверяет входящие запросы (authorization metadata). MPC-ноды проверяют, а twofa — нет.
- **Fix**: Auth interceptor для входящих запросов от Gateway. (gRPC-сетевое — **пропускаем**)

### M-15. Database password в DSN в config.yaml
- **Файл**: `config.yaml:7`
- `dsn: "postgres://twofa_user:twofa_pass@..."` — пароль в plaintext.
- **Fix**: env variable substitution для DSN.

---

## LOW

### L-01. Share data не zeroize при ошибке в retrieveShares
- **Файл**: `services/twofaService/retrieve_shares.go:51-69`
- При `ErrInsufficientShares` собранные доли остаются в памяти (caller's defer не выполняется, shares = nil).
- **Fix**: Зeroize перед return error.

### L-02. Panic на пустом secret в hotp()
- **Файл**: `crypto/totp/totp.go:107`
- `hotp()` паникует при пустом secret. DoS-вектор при баге в Shamir Combine.
- **Fix**: Return error вместо panic, или pre-check в callers.

### L-03. Share index из loop index, не из ответа MPC-ноды
- **Файл**: `services/twofaService/retrieve_shares.go:45`
- `Share{Index: byte(idx + 1)}` — из позиции в массиве clients, не из ответа ноды.
- **Fix**: MPC-нода должна возвращать share index в ответе для cross-validation.

### L-04. API handler imports конкретные service error types
- **Файл**: `api/twofa_service_api/setup.go:8`, `verify.go:9`, `disable.go:9`
- Handler импортирует `twofaService` для sentinel errors.
- **Fix**: Переместить sentinel errors в `domain` package.

### L-05. Bootstrap возвращает concrete types, не interfaces
- **Файл**: `bootstrap/bootstrap.go:25-27`
- `NewPGStorage` возвращает `*pgstorage.PGStorage`.
- **Fix**: Документировать как intentional или вернуть interface.

### L-06. Email формат не валидируется в Setup
- **Файл**: `api/twofa_service_api/setup.go:16`
- Проверка только на пустоту. Invalid email → невалидный provisioning URI.
- **Fix**: Базовая email validation.

### L-07. Backup code entropy 26.6 бит
- **Файл**: `services/twofaService/backup_codes.go:20-30`
- "xxxx-xxxx" = 10^8 = ~26.6 бит. Приемлемо с учетом bcrypt, но на нижней границе.

### L-08. Anemic domain models
- **Файл**: `models/models.go`
- `TwoFARecord` и `BackupCode` — чистые data structs без поведения. Допустимо для микросервиса этого размера.

### L-09. BackupCode model определен, но не используется для queries
- **Файл**: `models/models.go:13-18`
- Struct определен, но нет storage-метода для получения backup-кодов.

### L-10. `cmp.Or` для default metricsPort
- **Файл**: `cmd/app/main.go:64-67`
- `if metricsPort == 0 { metricsPort = 9101 }` → `cmp.Or(cfg.Server.MetricsPort, 9101)`.

### L-11. `cmp.Or` для default MPCTimeout
- **Файл**: `config/config.go:27-29`
- `if c.MPCTimeout == 0 { return DefaultMPCTimeout }` → `cmp.Or(c.MPCTimeout, DefaultMPCTimeout)`.

### L-12. Manual byte comparison в totp_test.go
- **Файл**: `crypto/totp/totp_test.go:94-101`
- Ручное побайтовое сравнение вместо `bytes.Equal`.

---

## Testing Issues (отдельно от кода)

### Отсутствующие тесты
- Нет тестов: `backup_codes.go`, `retrieve_shares.go`, `audit.go`, gRPC handlers, middleware, storage layers
- Нет error path тестов: config (non-existent file, malformed YAML), Disable (`DeleteBackupCodes` fail, `DeleteTwoFARecord` fail, OTP reuse), Verify (`GetTwoFARecord` DB error, `Combine` fail, `EnableTwoFA` fail)
- Нет direct unit tests: `evalPolynomial`, `lagrangeInterpolateAtZero`, `ValidateOTP` (time.Now-based API)

### Качество тестов
- `TestSetup_SharesZeroized` — не проверяет zeroization (comment в коде подтверждает)
- Timing-sensitive TOTP тесты (window boundary, уже были flaky — commit `10da858`)
- Excessive `makeAllMocksOptional` в disable/status тестах — mock expectations бесполезны
- Config test зависит от реального файла (`../config.yaml`)
- Status тесты — massive boilerplate duplication, нужен `newStatusSuite`
- `t.Parallel()` нигде не используется
- Table-driven tests underused (shamir roundtrips, shamir errors, totp non-six-digit)
- EventProducer всегда Optional — audit events не тестируются
- Error message assertions fragile (`strings.Contains(err.Error(), "...")`)
- `newSetupSuite` передает `nil` для `sessionStorage`
