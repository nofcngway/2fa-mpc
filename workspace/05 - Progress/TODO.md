# TODO

## Сервисы
- [x] Auth Service — полная реализация
- [x] TwoFA Service — полная реализация
- [x] MPC Node — полная реализация
- [ ] API Gateway — полная реализация

## Инфраструктура
- [ ] Docker Compose (все сервисы + PostgreSQL + Redis + Kafka + Prometheus + Grafana)
- [ ] Prometheus конфигурация (scrape targets)
- [ ] Grafana dashboards
- [ ] Kafka topics конфигурация

## Тестирование
- [x] Unit-тесты Shamir (split/combine, threshold)
- [x] Unit-тесты TOTP
- [x] Unit-тесты валидации паролей
- [x] Unit-тесты AES-256-GCM
- [ ] Интеграционные тесты Auth
- [ ] Интеграционные тесты TwoFA + MPC

## Документация ()
- [ ] Описание архитектуры
- [ ] Описание протоколов безопасности
- [ ] Диаграммы последовательностей
- [ ] Результаты тестирования
