# mTLS — взаимная аутентификация между сервисами

С Phase B (2026-05-10) все внутренние gRPC-каналы защищены mutual TLS. Этот документ описывает PKI, конфигурацию, чем mTLS дополняет существующий shared_secret и как добавить новый сервис.

## Архитектура каналов

```
Frontend → HTTPS (terminated externally) → Gateway
Gateway  → mTLS (TLS 1.3) → Auth     (gRPC port 9090)
Gateway  → mTLS (TLS 1.3) → TwoFA    (gRPC port 9091)
TwoFA    → mTLS (TLS 1.3) → MPC-1    (gRPC port 9200)
TwoFA    → mTLS (TLS 1.3) → MPC-2    (gRPC port 9201)
TwoFA    → mTLS (TLS 1.3) → MPC-3    (gRPC port 9202)
```

Каждый сервис одновременно server и client (кроме Gateway, который только client; и MPC-нод, которые только server).

## PKI

Все сертификаты подписаны одним dev-CA (`certs/ca.crt`). Для каждого сервиса один cert+key pair с EKU `serverAuth, clientAuth` — один cert используется и при приёме входящих, и при инициации исходящих соединений.

| Identity | SAN | Используется как |
|----------|-----|------------------|
| `auth` | `DNS:auth, DNS:localhost, IP:127.0.0.1` | server (Gateway → Auth) |
| `twofa` | `DNS:twofa, DNS:localhost, IP:127.0.0.1` | server (Gateway → TwoFA), client (TwoFA → MPC) |
| `mpc-node-1`, `mpc-node-2`, `mpc-node-3` | `DNS:mpc-node-N, ...` | server (TwoFA → MPC) |
| `gateway` | `DNS:gateway, DNS:localhost, IP:127.0.0.1` | client (Gateway → Auth/TwoFA) |

Сгенерировать вручную: `scripts/gen-certs.sh`. CA срок жизни 10 лет, leaf 825 дней.

**Автоматическая генерация в docker-compose:** init-контейнер `certgen` (alpine + openssl) запускается перед всеми сервисами через `depends_on: certgen: { condition: service_completed_successfully }`. Скрипт идемпотентен — пропускает существующие файлы. Принудительная регенерация:

```bash
rm -rf certs/
docker compose up
# или: docker compose run --rm certgen bash /work/scripts/gen-certs.sh --force
```

**Production:** генерация dev-CA скриптом — только для разработки и демо. В продакшне нужен managed PKI (Vault, cert-manager, частный CA) с ротацией ключей и offline custody CA.key.

## Конфигурация

Каждый сервис принимает TLS-секцию:

```yaml
tls:
  enabled: true
  cert_file: /certs/<service>.crt
  key_file: /certs/<service>.key
  ca_file: /certs/ca.crt
```

Env-overrides: `<SERVICE>_TLS_ENABLED`, `<SERVICE>_TLS_CERT_FILE`, etc.

Если `tls.enabled=false`, сервис стартует с insecure credentials и логирует громкий warning. Production обязан включать TLS — нет автоматического fail-fast, но warning виден в логах сразу при старте.

## Реализация

Per-сервис helper в `<svc>/internal/bootstrap/tls.go`:
- `loadServerTLSCredentials(cert, key, ca)` — для серверов (auth, twofa, mpc): требует client cert (`tls.RequireAndVerifyClientCert`), TLS 1.3 минимум.
- `loadClientTLSCredentials(cert, key, ca)` — для клиентов (gateway, twofa→mpc): валидирует server cert по CA.

Transport-decision в `<svc>/internal/bootstrap/<x>_transport.go`:
- `mpcTransportCreds` (twofa) / `clientTransportCreds` (gateway) — выбор между TLS и insecure на основе `cfg.TLS.Enabled`.

Adapter в `twofa/internal/adapters/mpcclient/client.go` — implementation `twofaService.MPCClient` через gRPC. Изолирует знание о protobuf от use-case-слоя.

Client interceptor в `twofa/internal/middleware/client_auth.go` — добавляет `authorization: <shared_secret>` в metadata исходящих запросов. **Это defense-in-depth, не основная защита** — основная защита это mTLS.

## Что НЕ покрывает mTLS

- **Replay внутри сессии** — защищено TLS sequence numbers.
- **Cross-session replay** — требует подделать TLS handshake = требует приватный ключ клиента. Защищено криптографией.
- **Авторизация end-user** — это уровень выше: Gateway проверяет JWT access-токен и извлекает user_id.
- **Ротация сертификатов** — не реализована автоматически. Operator должен перезапускать сервисы при обновлении сертификатов. Production-PKI должен это решать.
- **Cert revocation (CRL/OCSP)** — Go's crypto/tls не проверяет CRL по умолчанию. Если cert скомпрометирован, нужно перевыпустить CA.

## Тестирование

`twofa/internal/bootstrap/tls_test.go`:
- `TestLoadServerTLSCredentials` / `TestLoadClientTLSCredentials` — корректная загрузка с реальных файлов из `certs/`
- `TestLoadServerTLSCredentials_RejectsMissingFiles` — fail на пустых/несуществующих путях
- `TestMTLS_EndToEnd` — настоящий gRPC handshake поверх loopback TCP с твгенерированными сертификатами + Health.Check RPC

Тесты skip'аются если `certs/ca.crt` не существует.

## Добавление нового сервиса

1. Добавить SAN в `scripts/gen-certs.sh`: `gen_leaf newservice "DNS:newservice"`.
2. Перегенерировать сертификаты.
3. В config сервиса добавить `TLSConfig` секцию по образцу.
4. В bootstrap добавить `tls.go` (server и/или client helpers).
5. Wire в server/client setup.
6. В docker-compose добавить env vars + `./certs:/certs:ro` volume.
7. Cert SAN должен совпадать с docker DNS-именем.

## Связанные решения

- [ADR Log § ADR-011](../04%20-%20Decisions/ADR%20Log.md) — выбор mTLS вместо HMAC request signing
- [MPC Fault Tolerance](MPC%20Fault%20Tolerance.md) — MPC threshold под mTLS

## Связанные сервисы

- Auth
- TwoFA
- MPC Node
- Gateway
