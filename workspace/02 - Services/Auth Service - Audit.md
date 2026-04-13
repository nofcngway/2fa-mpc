# Auth Service — Аудит

**Дата:** 2026-04-13
**Scope:** Полный аудит `auth/` — ошибки, безопасность, архитектура, тесты, Modern Go

---

## CRITICAL (1)

### 1. Нет Recovery Interceptor — паника убивает процесс
- **Файл:** `internal/bootstrap/server.go:16-19`
- gRPC сервер имеет только `MetricsInterceptor` и `LoggingInterceptor`
- Необработанная паника в любом хэндлере крашит весь процесс
- **Решение:** Добавить `RecoveryInterceptor` первым в цепочке — ловит панику, логирует стектрейс, возвращает `codes.Internal`

---

## HIGH (7)

### 2. Нет rate limiting на Login/Register
- **Файлы:** `cmd/app/main.go`, `internal/middleware/`
- Ни на уровне auth-сервиса, ни в interceptor'ах нет ограничения частоты запросов
- При прямом доступе к gRPC-порту — неограниченный брутфорс (~345k попыток/день через bcrypt)
- **Решение:** Rate-limiting interceptor или сетевая изоляция (mTLS + network policy)

### 3. Нет трекинга неудачных попыток входа / блокировки аккаунта
- **Файл:** `internal/services/authService/login.go`
- Нет счётчика неудачных попыток по email, нет экспоненциального backoff
- **Решение:** Счётчик в Redis по email + TTL, блокировка после N неудач

### 4. LogoutAll без аутентификации вызывающей стороны
- **Файл:** `internal/api/auth_service_api/logout_all.go:12-29`
- Любой gRPC-клиент с сетевым доступом может отозвать все сессии любого `user_id`
- Комментарий `WR-03` говорит о Phase 9, но сейчас это DoS-вектор
- **Решение:** mTLS или service-to-service токены до экспозиции эндпоинта

### 5. `GracefulStop()` блокирует shutdown навечно
- **Файл:** `cmd/app/main.go:100-105`
- `shutdownCtx` с 30с таймаутом создаётся, но `grpcServer.GracefulStop()` не принимает контекст
- Long-running RPC → shutdown зависает бесконечно
- **Решение:**
```go
done := make(chan struct{})
go func() { grpcServer.GracefulStop(); close(done) }()
select {
case <-done:
case <-shutdownCtx.Done():
    grpcServer.Stop()
}
```

### 6. Нет валидации конфигурации
- **Файл:** `config/config.go:53-65`
- `Load()` парсит YAML и сразу возвращает. Пустой DSN, нулевые TTL, отсутствующие пути к ключам — рантайм-падения вместо fail-fast
- **Решение:** Метод `Validate() error` с проверкой всех required-полей

### 7. `AuditEvent`/`EventProducer` в неправильном пакете — инвертированная зависимость
- **Файл:** `internal/services/authService/audit.go`
- `bootstrap/kafka.go` импортирует `authService` ради `EventProducer` и `AuditEvent` — инфраструктура зависит от сервисного слоя
- **Решение:** Вынести в `domain/` или отдельный `events/`

### 8. `GetUserByEmail` возвращает `(nil, nil)` — мина для nil-pointer
- **Файл:** `internal/storage/pgstorage/user.go:29-42`
- Вместо sentinel error `ErrUserNotFound` возвращается `(nil, nil)`
- Каждый вызывающий должен помнить проверять `user == nil` при `err == nil`
- **Решение:** Sentinel error `domain.ErrUserNotFound`

---

## MEDIUM (11)

### 9. Access и refresh токены структурно неразличимы
- **Файл:** `internal/services/authService/jwt.go:24-73`
- Единственная разница — поле `TokenFamily` (omitzero). Refresh-токен пройдёт `ValidateToken` как access
- **Решение:** Добавить claim `token_type` (`"access"` / `"refresh"`) с валидацией

### 10. Нет максимальной длины пароля (bcrypt truncation + DoS)
- **Файл:** `internal/services/authService/password_validation.go`
- bcrypt обрезает на 72 байтах: `"A"*72+"B"` и `"A"*72+"C"` дают одинаковый хэш
- Мегабайтные пароли — CPU DoS
- **Решение:** Лимит 72-128 символов

### 11. Email enumeration через регистрацию
- **Файл:** `internal/api/auth_service_api/register.go:33`
- `codes.AlreadyExists` + `"user with this email already exists"` — позволяет проверить существование email
- **Решение:** Общий ответ или принять как tradeoff (нет email verification по ТЗ)

### 12. Ошибка `DeleteTokenFamily` при обнаружении кражи токена тихо проглатывается
- **Файл:** `internal/services/authService/refresh_token.go:29`
- `_ = s.sessionStorage.DeleteTokenFamily(...)` — при недоступности Redis украденная семья остаётся активной
- **Решение:** Как минимум `slog.Warn` при ошибке

### 13. Kafka `Async: true` — аудит-события теряются без логирования
- **Файл:** `internal/bootstrap/kafka.go:27-36`
- С `Async: true` ошибки доставки не возвращаются. Kafka лежит — полная тишина
- **Решение:** Выставить `ErrorLogger` на `kafka.Writer`

### 14. `PasswordValidationError` не реализует `Unwrap() []error`
- **Файл:** `internal/domain/errors.go:34-45`
- `errors.Is(err, domain.ErrPasswordTooShort)` вернёт `false` на `*PasswordValidationError`
- **Решение:** Добавить `Unwrap() []error { return e.Violations }`

### 15. `LogoutAll` не эмитит аудит-событие
- **Файл:** `internal/services/authService/logout_all.go:6-8`
- Security-critical операция без аудит-следа
- **Решение:** Добавить `PublishEvent` с `"user.logged_out_all"`

### 16. Login не аудирует неудачные попытки
- **Файл:** `internal/services/authService/login.go`
- Только успешные логины пишут аудит. Для брутфорс-детекции нужен `"user.login_failed"`
- **Решение:** Аудит-событие при `ErrInvalidCredentials`

### 17. `RedisStorage.Close()` ошибка игнорируется при shutdown
- **Файл:** `cmd/app/main.go:115-118`
- Kafka и metrics server проверяют ошибку Close, Redis — нет
- **Решение:** Проверить и залогировать ошибку

### 18. Нет nil-проверок в конструкторе `NewAuthService`
- **Файл:** `internal/services/authService/auth_service.go:41-59`
- 7 параметров без проверки. Nil `privateKey` → panic при первом вызове
- **Решение:** Nil-проверки в конструкторе, возврат ошибки или panic с ясным сообщением

### 19. Timing oracle для проверки существования пользователя
- **Файл:** `internal/services/authService/login.go:21-27`
- Несуществующий пользователь → ~0ms, существующий + неверный пароль → ~250ms (bcrypt)
- **Решение:** Dummy `bcrypt.CompareHashAndPassword` при user not found

---

## LOW (10)

### 20. `t.Context()` вместо `context.Background()` — 24 места
- **Все `_test.go` файлы**
- Go 1.24+ предоставляет `t.Context()` с автоотменой при завершении теста

### 21. `slices.Reverse` вместо ручного swap loop
- **Файл:** `internal/services/authService/password_validation.go:132-138`
- Функция `reverse()` → `slices.Reverse(runes)`

### 22. `cmp.Or` для дефолтного значения
- **Файл:** `cmd/app/main.go:67-69`
- `metricsPort := cmp.Or(cfg.Server.MetricsPort, 9100)`

### 23. `errors.Is` вместо `==` для sentinel error
- **Файл:** `cmd/app/main.go:76`
- `err != http.ErrServerClosed` → `!errors.Is(err, http.ErrServerClosed)`

### 24. `slices.ContainsFunc` в тестах
- **Файл:** `internal/services/authService/password_validation_test.go:225-232`
- Ручной цикл поиска → `slices.ContainsFunc`

### 25. Ошибки в Login не оборачиваются контекстом
- **Файл:** `internal/services/authService/login.go`
- В отличие от Register, Login возвращает сырые ошибки без `fmt.Errorf("...: %w", err)`

### 26. LoggingInterceptor логирует `"error": null` для успешных вызовов
- **Файл:** `internal/middleware/interceptors.go:33`
- **Решение:** Условное включение поля error

### 27. Дублирующийся тест `TestLogoutAll_ReturnsNilOnSuccess`
- **Файл:** `internal/services/authService/logout_all_test.go:67-75`
- Дублирует `TestLogoutAll_Success` — один удалить

### 28. Нет тестов для gRPC handler layer
- Слой `internal/api/auth_service_api/` — маппинг domain-ошибок → gRPC-кодов — полностью без тестов

### 29. `TestJWT_ParseToken_RejectsHS256_AlgorithmConfusion` содержит мёртвый код
- **Файл:** `internal/services/authService/jwt_test.go:90-121`
- `rsa.EncryptPKCS1v15` и `jwt.ParseRSAPublicKeyFromPEM(nil)` — мёртвый код
- Тест работает случайно, не проверяет реальную атаку algorithm confusion

---

## Итого

| Severity | Кол-во |
|----------|--------|
| CRITICAL | 1 |
| HIGH | 7 |
| MEDIUM | 11 |
| LOW | 10 |
| **Всего** | **29** |

## Топ-5 для немедленного исправления
1. Добавить **RecoveryInterceptor** (CRITICAL — crash protection)
2. Добавить **валидацию конфига** с fail-fast при старте
3. Исправить **GracefulStop** с таймаутом
4. Добавить claim **`token_type`** в JWT (access vs refresh)
5. Вынести **`EventProducer`/`AuditEvent`** в `domain/` (fix inverted dependency)
