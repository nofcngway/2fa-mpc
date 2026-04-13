# Обзор архитектуры

## Система
Двухфакторная аутентификация с распределенным хранением TOTP-секретов через протокол Shamir Secret Sharing (2-of-3).

## Компоненты

```
User → Frontend (Next.js) → HTTPS → API Gateway (Go, REST→gRPC)
                                        ├── Auth Service (Go, gRPC)
                                        └── TwoFA Service (Go, gRPC)
                                              ├── MPC Node 1
                                              ├── MPC Node 2
                                              └── MPC Node 3
```

## Инфраструктура
| Компонент | Версия | Роль |
|-----------|--------|------|
| Go | 1.26.2 | Язык всех backend-сервисов |
| PostgreSQL | latest | Персистентное хранение |
| Redis | 8.6.2 | Refresh-сессии, rate limiting |
| Kafka | 4.1.2 | Асинхронные события аудита |
| Prometheus | latest | Сбор метрик |
| Grafana | latest | Визуализация метрик |

## Принципы
- Clean Architecture (handler → service → repository)
- DI через bootstrap-фабрики
- gRPC между сервисами, REST только на входе (Gateway)
- Секреты никогда не хранятся целиком — только доли (Shamir 2-of-3)
- Шифрование долей at-rest (AES-256-GCM)
