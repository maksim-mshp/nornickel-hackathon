# 09. Спецификация микросервисов

**Все Go-сервисы — один Go-проект** (единый `go.mod`): разные точки входа в `cmd/<service>/main.go`, bounded contexts — пакеты `internal/<service>/…`, общее — `internal/platform/…`; один Dockerfile с `ARG SERVICE` собирает любой бинарь ([03-architecture.md](03-architecture.md) §9). Деплой-единицы остаются отдельными контейнерами — «микросервисность» в границах контекстов и независимом масштабировании, а не в множестве репозиториев.

Общие свойства всех Go-сервисов: Go 1.25.x (в `go.mod` — строго `go 1.25.0`); чистая архитектура + DDD (эталонная структура — [03-architecture.md](03-architecture.md) §9); golangci-lint с `modernize` ([16-dev-environment.md](16-dev-environment.md)); **вся конфигурация — YAML** (каркас `internal/platform/config` на koanf v2): `configs/base/<service>.yml` (дефолты, в git) + оверлей окружения `configs/<env>/<service>.yml` + `configs/secrets.yml` (merge последним; в .gitignore, шаблон — `secrets.yml.example`); env-переменные и `.env` не используются; путь до конфига — флаг `--config` (по умолчанию `configs`), выбор окружения — `--env dev|demo|prod`; валидация схемы конфига при старте (fail-fast с внятной ошибкой); OTel-трейсинг/метрики/логи (slog JSON, `request_id`/`trace_id` сквозные, в NATS-заголовках `traceparent`); health-пробы `/healthz` (liveness) и `/readyz` (зависимости); graceful shutdown ≤30 c (drain NATS-консьюмеров, отмена контекстов); Dockerfile multi-stage на базе alpine, nonroot; лимиты ресурсов и реплики — [12-deployment.md](12-deployment.md) §3.1.

Python-сервисы: Python 3.13 (uv), gRPC-сервер (`grpcio`), ruff+mypy, слои `domain/app/ports/adapters` ([16-dev-environment.md](16-dev-environment.md) §2), тот же OTel/health-контракт; **конфигурация — те же YAML** из `configs/` (pydantic-settings с YAML-источником, та же схема base+оверлей+secrets.yml); образы `python:3.13-slim` (alpine несовместим с ML-зависимостями).

---

## 1. kmap-gateway (Go) — публичный API

- **Назначение:** единственная точка входа UI/интеграций. REST (OpenAPI 3.1 из аннотаций **swag v2**; спека публикуется на `/openapi.json`, Swagger UI на `/docs`), SSE-стриминг ответов, WebSocket не используем.
- **Auth/аудит — устаревшее требование ТЗ (demo-режим as-is, ADR-6): не приоритет, дальше не развивается:**
  - **AuthN:** OIDC (Keycloak 26.6): валидация JWT (JWKS-кэш), маппинг claims → `Principal{user_id, roles[], doc_access}`. Для демо — статические токены-роли (`auth.mode: demo` в YAML).
  - **AuthZ:** middleware RBAC по таблице «роль → операции» ([01-brief.md](01-brief.md) §2.2); установка RLS-контекста запроса (`SET LOCAL app.doc_access`, `app.user_id`) через передачу principal в gRPC-метаданных вниз.
  - **Аудит:** каждое действие (search/view/export/edit/login) → `kmap.audit.v1.*` (fire-and-forget в JetStream) + синхронная запись критичных (export, fact_edit) в PG.
- **API-поверхность:** см. [10-contracts.md](10-contracts.md) §2.
- **Зависимости (gRPC):** answer, search, catalog, ingest. **Масштабирование:** stateless; реплики по RPS (compose scale).

## 2. kmap-ingest (Go) — приём и реестр документов

- Multipart-приём (стрим в MinIO, без буферизации в память), sha256-дедуп, версии, манифест-батчи, URL-fetcher (allowlist).
- Маппинг структурированных источников (CSV/XLSX каталога экспериментов; справочники) → прямые gRPC-вызовы catalog.
- Статусы конвейера (`ops.ingest_jobs`) и их выдача (`GET /v1/documents/{id}/status`).
- Транзакционный outbox + relay-паблишер (общий пакет `internal/platform/outbox`).
- **Масштабирование:** stateless; загрузки шардируются по document_id.

## 3. kmap-parse (Python) — DocIR

- Консьюмер `doc.v1.registered`; Docling 2.x (layout, reading order, TableFormer); выход DocIR (`docir/1`) в MinIO; язык (fasttext), тип документа (правила+zero-shot).
- OCR — `parse.ocr: off | auto` в YAML (off по умолчанию: организаторы гарантируют текстовый слой).
- Пул воркеров = CPU cores; тяжёлые документы не блокируют очередь (per-msg ack_wait 15m, max_ack_pending=workers).
- **Масштабирование:** горизонтально (замер: ~3–8 с на типовой PDF-отчёт 30 стр. на 4 vCPU).

## 4. kmap-extract (Python) — кандидаты знаний

- Консьюмер `doc.v1.parsed`: чанкинг → эмбеддинги (батчи в kmap-embed) → numcore (детерминированный, [06-extraction.md](06-extraction.md)) → LLM-извлечение (через kmap-llm, схемы strict) → ExtractionBundle в MinIO.
- Версии: `numcore-X.Y.Z`, `prompts/<task>@<ver>` — в bundle.quality и в каждом факте.
- Матрица моделей по задачам и лимит параллельных LLM-вызовов — `configs/*/llm-routes.yml`.
- **Масштабирование:** горизонтально; полная переиндексация 10⁴ документов ≤8 ч на 4 репликах (с локальным vLLM).

## 5. kmap-embed (Python) — эмбеддинги и rerank

- gRPC: `Embed(texts[], mode=dense|dense+sparse)` → f32[1024] (+sparse map), батчинг 64, микробатч-агрегатор 20 мс; `Rerank(query, passages[])` → scores. В remote-режиме эмбеддинги идут через официальный `openai` SDK (Python, зависимость `openai>=1.40`); rerank — `bge-reranker-v2-m3` через DO `/v1/rerank` прямым HTTP (rerank вне OpenAI-спеки, у SDK нет метода); офлайн-фолбэк rerank — локальный token-overlap (Jaccard).
- Модели: BAAI/bge-m3, BAAI/bge-reranker-v2-m3; device auto (CUDA→CPU); прогрев при старте (readyz после загрузки весов); `embed.backend: remote | torch | onnx-int8` в YAML: **remote — дефолт** (bge-m3/reranker через DO Gradient, $0.02/$0.01 за 1M, проверено 02.07; сервис — тонкий gRPC-фасад ~256 МБ RAM); torch/onnx-int8 — локальные офлайн-режимы (onnx-int8 — для слабых машин).
- LRU-кэш эмбеддингов запросов (по sha256 текста).
- **Масштабирование:** вертикально (GPU) или репликами (CPU); p95 embed-запроса ≤80 мс (GPU) / ≤400 мс (CPU, батч 1×512 ток.).

## 6. kmap-catalog (Go) — ядро домена

- gRPC-API: CommitExtraction(bundle_uri), ResolveEntities(names[]) (префетч для парсера запросов), MergeEntities, UpdateFactStatus, UpsertSeed (справочники), SuggestAliases-приём.
- Консьюмер `doc.v1.extracted` → CommitExtraction.
- Entity resolution каскад (детерминированный, [05-ingestion.md](05-ingestion.md) §6); очередь ревью (pending_review) и админ-операции экспертов (всё в fact_history + audit).
- Инварианты фактов ([04-data-model.md](04-data-model.md) §4) — единственное место enforcement.
- **Масштабирование:** 2 реплики (active-active; сериализация конфликтов на уровне PG-констрейнтов и advisory-lock по document_id).

## 7. kmap-llm (Go) — LLM-шлюз

- gRPC: `Complete(task, payload, schema_ref, stream)` → валидированный JSON/стрим токенов.
- Роутинг задач → upstream+модель (`configs/*/llm-routes.yml`, `default_provider: yandex`; ключи — `configs/secrets.yml`): OpenAI-совместимые endpoint'ы (Yandex AI Studio `ai.api.cloud.yandex.net` / vLLM on-prem). Failover-цепочки провайдеров через `fallback_providers` (по умолчанию `[do_gradient]`: Yandex → DigitalOcean тем же `openai-go`, с переводом слагов моделей — [15-resources.md](15-resources.md) §1.3), circuit breaker (gobreaker), ретраи с джиттером (только идемпотентные), таймауты по задаче.
- JSON Schema-валидация ответа + до 2 repair-попыток (с фидбеком ошибок схемы); structured outputs / guided decoding, если upstream поддерживает (vLLM — поддерживает).
- Бюджеты: токен-квоты per-task/per-day, конкуренция per-upstream (semaphore), очередь batch-задач (judge) с приоритетом ниже интерактивных.
- Кэш: PG-таблица `ops.llm_cache` (ключ sha256(model+prompt+schema), TTL по задаче) — судья/извлечение переиспользуются при реплеях (экономия ресурсов — критерий жюри).
- Журнал вызовов (без содержимого при `llm.log_prompts: false` — ИБ-режим; метаданные всегда: task, model, tokens, latency, valid).
- **Масштабирование:** stateless, 2+ реплики.

## 8. kmap-search (Go) — retrieval

- gRPC: `Search(QueryPlan) → EvidencePack`; `EgoGraph(entity, depth≤3, top_n)`; `Entities/Experiments/Experts list+get` (браузинг для UI).
- Три канала ∥ (SQL-факты, pgvector, FTS) → RRF → rerank (kmap-embed) → сборка EvidencePack (+consensus/contradictions/gaps/experts из epi.*) — [07-query-pipeline.md](07-query-pipeline.md) §2.
- Read-only: отдельный read-DSN (сейчас — тот же инстанс; при выносе PG — streaming-реплика без изменения кода).
- **Масштабирование:** stateless; узкое место — PG, первый шаг роста описан в [12-deployment.md](12-deployment.md) §7.

## 9. kmap-answer (Go) — оркестратор ответа

- gRPC stream: `Ask(question, filters, principal) → события plan/evidence/answer.delta/answer.done`.
- Двухконтурный парсер QueryPlan (LLM + rules, кросс-чек чисел), декомпозиция comparison, вызов search, синтез через kmap-llm (стрим), **numeric guard**, кэш ответов + инвалидация по событиям.
- Пресеты вопросов (Q1–Q6) — конфиг для UI.
- **Масштабирование:** stateless; реплики по числу активных SSE.

## 10. kmap-epistemic (Go) — эпистемика

- Консьюмеры `facts.v1.committed`/`epistemic.v1.cluster-dirty`; шардирование воркеров по hash(ckey); батч-очередь judge через kmap-llm; ночной полный пересчёт (cron-лидер через PG advisory lock); drift-детектор.
- gRPC: `GetCoverage(matrix-запрос для heatmap)`, `GetContradictions(cluster/entity)`, `DecideContradiction(эксперт)`.
- **Масштабирование:** 2 реплики; пересчёт партиционирован по ckey.

## 11. kmap-ui (Next.js + TypeScript)

Полная спецификация — **[17-frontend.md](17-frontend.md)**: дизайн-система «Полярная ночь/Электролит» (токены, типографика Unbounded/Golos Text/JetBrains Mono, motion, фирменные компоненты — штамп-провенанс, guard-пломба, консенсус-спектр), стек (Next.js 15 App Router, Tailwind v4, shadcn/Radix, TanStack Query+Table, Cytoscape.js, SSE-клиент с реконнектом) и девять экранов: Research Workspace (`/`), паспорт сущности, каталог экспериментов, карта покрытия, эксперты, реестр документов, очередь ревью, словари, сохранённый ответ. Интерфейс — только RU (переключатель языка удалён; мультиязычность RU/EN — на уровне корпуса и поиска, не UI), темы night/protocol.

## 12. kmap-eval (Python) — качество

CLI + CI-джоб: прогон gold-набора через публичный API, расчёт метрик ([13-evaluation.md](13-evaluation.md)), сравнение с базовой линией прошлого прогона, markdown-отчёт в артефакты CI; красный статус при деградации сверх порогов. Также офлайн-оценка numcore на размеченных фактах (юнит-уровень, без сервисов).

## 13. kmap-notify (Go, Phase 2)

Подписки пользователя на сущности/темы; консьюмер `facts.v1.committed`+`epistemic.v1.updated`; матчинг подписок; дайджест (email/webhook). В MVP — только модель данных подписки (таблица) и событие, без доставки.

---

## Сводная таблица

| Сервис | Язык | Вход | Выход | Стейт |
|---|---|---|---|---|
| gateway | Go | REST/SSE | gRPC fan-out, audit events | нет |
| ingest | Go | REST(gw), манифесты | MinIO, PG(core), events | нет |
| parse | Py | events | MinIO DocIR, events | нет |
| extract | Py | events | MinIO bundle, events | нет |
| embed | Py | gRPC | векторы/скоры | модель в RAM |
| catalog | Go | gRPC, events | PG(kg), events | нет |
| llm | Go | gRPC | upstream HTTP | кэш в PG |
| search | Go | gRPC | EvidencePack | нет (read-replica) |
| answer | Go | gRPC stream | SSE-события | кэш в PG |
| epistemic | Go | events, gRPC | PG(epi), events | нет |
| ui | TS | браузер | REST/SSE | нет |
| eval | Py | CLI/CI | отчёты | нет |
