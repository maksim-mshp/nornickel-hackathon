# 12. Развёртывание, окружения, CI/CD

**Решение о деплое (зафиксировано 02.07.2026):** целевой контур — **одна Ubuntu VM (8 vCPU / 32 ГБ RAM), Docker Compose v2, без Kubernetes**. Уровень «enterprise» обеспечивается не оркестратором, а дисциплиной: healthchecks, лимиты ресурсов, пин образов, TLS, бэкапы с репетицией восстановления, наблюдаемость, деплой-скрипт с откатом. Kubernetes — отложенный путь масштабирования (§7), архитектура его не требует и не блокирует.

## 1. Монорепозиторий

```
nornickel-hackathon/
├── AGENTS.md, .agents/           # управление и проектная документация
├── .tmp/                         # ВСЕ временные файлы (в .gitignore)
├── contracts/{proto,openapi,events}/
├── go.mod                        # единый Go-модуль всего бэкенда
├── cmd/                          # точки входа Go-сервисов: gateway|ingest|catalog|llm|search|answer|epistemic
├── internal/                     # bounded contexts (слои domain/app/ports/adapters) + platform (общее)
├── configs/                      # ВСЯ конфигурация — YAML: base/ (дефолты) + dev/ demo/ prod/ (оверлеи)
│                                 # + secrets.yaml (ключи; в .gitignore, шаблон secrets.yaml.example)
├── Dockerfile                    # один на все Go-сервисы (ARG SERVICE)
├── services/                     # Python-сервисы (отдельные uv-проекты): parse/ extract/ embed/ eval/
├── ui/                           # Next.js
├── db/{migrations,seeds}/        # golang-migrate SQL (NNNN_name.up/.down.sql) + YAML-справочники
├── prompts/                      # версионированные промпты
├── eval/                         # gold-set, разметка, скрипты
├── deploy/
│   ├── compose/                  # dev/demo: docker-compose.yaml, профили core|llm-local|obs
│   └── vm/                       # прод на VM: compose.prod.yaml, caddy/, backup/, systemd/, deploy.sh
├── ops/{runbooks,dashboards,alerts,k6}/
├── Taskfile.yml                  # единые входы: task dev|demo|test|lint|gen|seed|eval|tools|deploy
└── .github/workflows/            # CI
```

Конвенция коммитов: **Conventional Commits с описанием на русском** — `<type>: <описание по-русски>` (типы: `feat`, `fix`, `docs`, `refactor`, `test`, `perf`, `ci`, `chore`). Пример: `feat: добавлен метод для взаимодействия с LLM`.

## 2. Dev/demo-контур (локально и на хакатоне)

Один `task demo` поднимает всё; профили compose: `core` (обязательное), `llm-local` (vLLM, только при наличии GPU), `obs` (наблюдаемость). Настройки — в `configs/demo/*.yaml` (upstream LLM: `llm-routes.yaml`; `parse.ocr: off`; `embed.backend: remote` и т.д.); ключи — в `configs/secrets.yaml` (в .gitignore); каталог `configs/` монтируется в контейнеры read-only. `.env` не используется вовсе. Авторизация в демо — `auth.mode: demo` (статические токены-роли, полный RBAC/RLS без внешнего IdP, [03-architecture.md](03-architecture.md) ADR-6); Keycloak — в прод-контуре.

Резервный сценарий питча (риск из RFC §32.2): датасет проиндексирован заранее (`task seed-demo`), live-ingestion показываем на одном маленьком документе, при сбое — пропускаем.

### 2.1. Демо-VM: Ubuntu 8 vCPU / 16 ГБ RAM / ~37 ГБ свободного диска (профиль `demo-slim`)

Подтверждённая конфигурация для развёртывания демо (compose, без Swarm). Отличия от прод-профиля §3:

- **Выключены**: obs-стек (−2.5 ГБ RAM; при желании можно включить — диск позволяет), caddy (порты gateway/ui напрямую), vLLM (LLM — только внешний upstream).
- **Эмбеддинги — remote** (`embed.backend: remote` в `configs/demo/embed.yaml`): bge-m3 + rerank через DO Gradient ($0.02/$0.01 за 1M, проверено 02.07) — веса не грузятся, embed-контейнеру хватает 256 МБ; офлайн-фолбэк — `onnx-int8` (+~2 ГБ RAM).
- Бюджет RAM (лимиты compose): PG 3 ГБ (shared_buffers 1 ГБ) · embed 0.3 (remote) · parse 2 · extract 1 · 7×Go по 128 МБ ≈1 · UI 0.4 · NATS 0.5 · MinIO 0.5 → **≈8.7 ГБ**, остальное — ОС/page cache. Swap-файл 4 ГБ — обязателен (страховка OOM на пиках parse).
- Бюджет диска (из ~37 ГБ свободных): образы ~6–8 (Python-сервисы строятся от **общего базового слоя** с torch — дедуплицируется) · данные PG+MinIO+NATS до ~20 · логи ~0.5 → запас ≥8 ГБ. Гигиена стандартная: лимит JetStream-стримов 4 ГБ, MinIO lifecycle на bundles, `docker system prune -af --volumes=false` в деплое, ротация логов.
- **Потолок корпуса: ~10–15 тыс. документов** — датасет хакатона помещается с большим запасом. Ограничение сместилось с диска на RAM PG; для полного прод-корпуса — профиль §3 (32 ГБ RAM).

## 3. Прод-контур: одна Ubuntu VM (8 vCPU / 32 ГБ)

### 3.1. Состав (deploy/vm/compose.prod.yaml)

Тот же compose-файл, что и demo, + прод-оверлей: пин образов по digest, лимиты, рестарт-политики, TLS.

| Группа | Контейнеры | Лимит RAM | Примечание |
|---|---|---|---|
| Вход | caddy (reverse proxy, TLS, gzip) | 128 МБ | авто-TLS (Let's Encrypt) или корп. сертификат |
| Go-сервисы ×7 | gateway, ingest, catalog, llm, search, answer, epistemic — бинарии из одного Dockerfile (`ARG SERVICE`) | по 256 МБ (~1.8 ГБ) | stateless, `deploy.replicas` при необходимости |
| Python | parse | 3 ГБ | пик на больших PDF |
| Python | extract | 2 ГБ | |
| Python | embed (bge-m3 + reranker) | 4 ГБ локально (fp32) **или** 256 МБ в remote-режиме | remote через DO — дефолт демо; локально — офлайн-опция |
| UI | ui (Next standalone) | 384 МБ | |
| Данные | postgres:18+pgvector | 8 ГБ (shared_buffers 4 ГБ) | named volume на SSD |
| Данные | nats (JetStream) | 1 ГБ | file store, volume |
| Данные | minio (single) | 1 ГБ | volume; versioning на kmap-raw |
| Auth | keycloak | 1.5 ГБ | OIDC (ADR-6); realm импортируется из `configs/keycloak/realm.json` |
| Набл. | otel-collector, prometheus, grafana, loki, tempo | ~2.5 ГБ | профиль obs — включён в проде |
| **Итого** | | **≈ 25 ГБ** | запас ~7 ГБ на page cache ОС |

CPU: интерактивный путь (search/answer/gateway) занимает <2 vCPU при 50 RPS; parse/extract/embed утилизируют остальное при индексации (фоново, nice через `cpus:`-лимиты: embed ≤4, parse ≤3). **LLM-инференс на VM не размещаем** (нет GPU): upstream — реестр моделей организаторов / API владельца; профиль `llm-local` на VM выключен.

### 3.2. Enterprise-дисциплина на одной машине

- **Изоляция**: два docker-network (`edge`: caddy+gateway+ui; `internal`: всё остальное); наружу открыты только 80/443 (+SSH); PG/NATS/MinIO не публикуют порты на хост.
- **Устойчивость**: `restart: unless-stopped`; healthchecks у всех; зависимость `depends_on: condition: service_healthy`; `docker compose up -d --wait` в деплое; логи json-file с ротацией (`max-size: 50m, max-file: 3`) + отгрузка в Loki.
- **systemd-unit** `kmap.service` (Type=oneshot + `docker compose up -d`) — автостарт после ребута VM; unattended-upgrades для security-патчей ОС; fail2ban + ufw (80/443/SSH).
- **Конфигурация**: `configs/prod/*.yaml` (в git, ревьюится как код); **секреты** — только `configs/secrets.yaml` вне git (mode 600 на VM, доставляется деплой-скриптом отдельно от репо); ключи LLM — только там.
- **Бэкапы** (cron-контейнер `backup`): `pg_dump -Fc` каждые 6 ч + непрерывный WAL-архив (wal-g) → MinIO-бакет `kmap-backups` **и** офф-VM копия (rclone на внешний S3/NAS — обязательна: VM = единая точка отказа); volume-снапшоты NATS/MinIO ежедневно; restore-репетиция — ежемесячно на dev-машине (`task restore-drill`).
- **Деплой** (`deploy/vm/deploy.sh`, вызывается CI по SSH): `docker compose pull` → поочерёдный `up -d --wait` по сервисам (gateway последним) → smoke-тест (5 gold-вопросов) → при провале `docker compose rollback` = up с предыдущими digest'ами (хранятся в `releases.log`). Даунтайм отдельного сервиса — секунды; полный zero-downtime не обещаем (честный SLO §11-nfr).
- **Watchdog**: alertmanager → Telegram/email; node-exporter + cadvisor в профиле obs.

### 3.3. Docker Swarm (опция, не default)

Если захотим rolling-updates и нативные secrets на той же машине — `docker swarm init` (single-node) и тот же compose как stack (`deploy:`-секции уже совместимы). Выгоды на одной ноде малы; включаем только если деплой-скрипт §3.2 станет тесен. Multi-node Swarm не планируем — следующая ступень сразу k8s (§7).

## 4. CI/CD (GitHub Actions)

```
PR:    lint (golangci-lint с modernize, ruff, eslint) → buf lint+breaking → unit-тесты (Go race, pytest)
       → golden-тесты numcore → integration (testcontainers: PG+NATS+MinIO)
       → build образов (buildx) → trivy scan
       → oasdiff (REST breaking) → swag init --v3.1 (актуальность спеки: diff = fail)
main:  всё выше + e2e (compose up + smoke gold-set 6 вопросов) → push registry (+SBOM syft)
       → deploy на stage-VM (ssh + deploy.sh) → kmap-eval полный прогон → отчёт в PR
tag:   deploy на prod-VM (manual approval в environment) → smoke на prod
nightly: k6-нагрузка на stage, полный eval, drift-отчёт эпистемики; еженедельно — restore-репетиция
```

Версии образов: `ghcr.io/<org>/kmap-<svc>:<git-sha>` + семверный тег на релизе. Каждый образ — на базе alpine, где возможно (Go, UI; Python ML — `python:3.13-slim`), nonroot, healthcheck. Деплой-секреты: `gh secret set` (SSH-ключ stage/prod, digest-реестр).

## 5. Окружения

| Параметр | dev (локально) | demo-VM (Ubuntu 8 vCPU/16 ГБ/37 ГБ, §2.1) | prod-VM (Ubuntu 8 vCPU/32 ГБ) |
|---|---|---|---|
| Запуск | `task dev` (hot-reload) | `task demo` (профиль demo-slim) + systemd | systemd + deploy.sh из CI |
| Конфиги | `configs/base` + `configs/dev` | + `configs/demo` | + `configs/prod` |
| PG / NATS / MinIO | по 1 контейнеру | то же | то же + лимиты, WAL-архив, офф-VM бэкап |
| Auth | `auth.mode: demo` (токены-роли) | `auth.mode: demo` | Keycloak/OIDC (`auth.mode: oidc`) + корп. IdP federation позже |
| LLM | реестр организаторов / API владельца | то же (без локального LLM) | реестр / API владельца (без локального LLM) |
| Embeddings | fp32 или int8 | **ONNX int8** | fp32 (RAM позволяет) |
| TLS / вход | localhost | порты напрямую + ufw | caddy :443, ufw |
| Наблюдаемость | профиль obs по желанию | выключена (логи docker + ротация) | обязательна + алерты |
| Данные | заглушка-корпус | датасет хакатона (до ~10–15 тыс. док.) | полный корпус |

## 6. Definition of Done релиза

1. CI зелёный, включая golden numcore и smoke gold-set.
2. `kmap-eval` на stage: метрики не ниже базовой линии ([13-evaluation.md](13-evaluation.md) §4).
3. Нет критических trivy-находок; SBOM приложен.
4. Миграции обратно-совместимы (expand-фаза), откат по `releases.log` проверен.
5. Дашборды/алерты обновлены при новых метриках.

## 7. Путь масштабирования (когда одной VM станет мало)

Триггеры: устойчиво >70% CPU в интерактивном пути, RAM-давление, потребность в HA без даунтайма, >10⁵ документов. Шаги по порядку, без переписывания кода (всё уже за интерфейсами/env): (1) вынести PG на отдельную VM + реплика (streaming), search — на реплику; (2) вынести embed/parse/extract на воркер-VM (NATS уже разделяет их с горячим путём); (3) полноценный k8s (Helm-чарты, CloudNativePG, NATS chart, HPA/KEDA) — заготовки описаны в истории этого документа и восстанавливаются при необходимости. Compose-файлы и образы переиспользуются как есть.
