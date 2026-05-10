# Load Tests

Нагрузочное тестирование MPC-2FA через k6. Сценарии выполняются против полного docker-compose стека (с mTLS включённым) и измеряют latency / throughput / error rate всех ключевых эндпоинтов.

## Сценарии

| Файл | Что измеряется | Bottleneck-кандидат |
|------|----------------|---------------------|
| `k6/login.js` | POST `/api/v1/auth/login` | bcrypt cost=12 — single-hash CPU |
| `k6/setup-2fa.js` | POST `/api/v1/2fa/setup` | TOTP secret + Shamir + 3 параллельных gRPC + 10 параллельных bcrypt (Phase A optimization) |
| `k6/verify-2fa.js` | POST `/api/v1/2fa/verify` | retrieveShares (2-of-3 first-2-wins) + Shamir combine + Redis (rate limit + OTP reuse) |
| `k6/mixed.js` | 70% verify + 20% login + 10% setup | Реалистичный production-like mix |

Каждый сценарий ramping-VUs: 15s warm-up → 45-60s steady → 15s ramp-down.

## Как запустить

Предполагается, что docker compose стек уже поднят (`make up-build` в корне).

```bash
# Из корня проекта:
docker compose -f docker-compose.yml -f loadtest/docker-compose.loadtest.yaml \
  --profile loadtest \
  run --rm k6 run /scripts/login.js

# Тоже самое для остальных сценариев:
... run /scripts/setup-2fa.js
... run /scripts/verify-2fa.js
... run /scripts/mixed.js
```

Или через Makefile target из корня:

```bash
make load-login
make load-setup
make load-verify
make load-mixed
make load-all     # все сценарии последовательно
```

Результаты:
- k6 печатает text summary в stdout (p50/p95/p99/max, RPS, error rate)
- JSON summary сохраняется в `loadtest/results/summary.json`
- Метрики каждого сервиса доступны в Prometheus (http://localhost:9190) и Grafana (http://localhost:3001) во время теста и после

## Сбор Prometheus-метрик

Во время теста в Grafana доступны live-графики через дашборд `mpc-2fa-overview`. Полезные PromQL для post-mortem анализа:

```promql
# RPS по эндпоинтам Gateway
sum(rate(http_requests_total{job="gateway"}[1m])) by (path, status)

# p99 latency по сервисам
histogram_quantile(0.99,
  sum(rate(grpc_request_duration_seconds_bucket[1m])) by (le, service))

# Error rate
sum(rate(http_requests_total{job="gateway",status=~"5.."}[1m]))
  / sum(rate(http_requests_total{job="gateway"}[1m]))

# CPU нагрузка контейнеров (требует cAdvisor — не входит в текущий стек)
```

## Ограничения

- Тесты работают на dev-стеке (общий хост, общий Docker daemon, single-node все сервисы). Цифры показывают **порядок** производительности и **относительные** улучшения, не absolute SLA.
- `verify-2fa.js` имеет ограничение по rate-limit'у TwoFA: 5 verify за 5 минут на user_id. Поэтому сценарий использует пул из 40 аккаунтов и `sleep(2)` между итерациями.
- TOTP-окно 30 секунд + защита от replay означают, что один аккаунт может верифицироваться максимум 1 раз за окно. Большой пул аккаунтов это смягчает.

## Отчёт

См. [`REPORT.md`](REPORT.md) — результаты конкретных запусков, анализ узких мест, рекомендации по масштабированию.
