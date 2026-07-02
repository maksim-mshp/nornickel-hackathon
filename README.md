# R&D Knowledge Map (kmap)

Единая карта знаний R&D для горно-металлургической отрасли: поисково-аналитическая система с проверяемым числовым ядром, эпистемическим слоем (консенсус/противоречия/пробелы) и графом институциональной памяти. Трек «Научный клубок», Норникель AI Science Hack 2026.

Проектная документация — [AGENTS.md](AGENTS.md) и [.agents/](.agents/README.md).

## Быстрый старт

```
task infra      # PostgreSQL 18 + pgvector, NATS JetStream, MinIO
task migrate    # схема БД
task dev        # инфраструктура + миграции
task demo       # полный демо-контур в docker
```

Секреты: скопируйте `configs/secrets.yml.example` в `configs/secrets.yml` и заполните ключ LLM API.

## Структура

- `cmd/` — точки входа Go-сервисов (gateway, ingest, catalog, llm, search, answer, epistemic)
- `internal/` — bounded contexts (domain/app/ports/adapters) + platform
- `contracts/` — proto/OpenAPI/события
- `configs/` — вся конфигурация (YAML, base + оверлеи окружений)
- `db/migrations` — golang-migrate SQL
- `services/` — Python-сервисы (parse, extract)
- `ui/` — Next.js фронтенд
