# 11. Нефункциональные требования: нагрузка, отказоустойчивость, безопасность, наблюдаемость

## 1. Целевые SLO (прод)

Контур — одна Ubuntu VM (8 vCPU/32 ГБ), Docker Compose ([12-deployment.md](12-deployment.md)); SLO ниже — честные для single-host, путь к HA зафиксирован в [12](12-deployment.md) §7.

| Показатель | Цель | Как обеспечиваем |
|---|---|---|
| Доступность API (месяц) | 99.5% (single-host потолок) | systemd-автостарт, healthchecks + restart-политики, деплой с откатом по digest, алерты |
| `/v1/ask` first token | p95 ≤ 1.5 c | evidence раньше синтеза; rules-fallback парсера |
| `/v1/ask` полный ответ | p95 ≤ 5 c (НФТ ТЗ: 3–5 c — **желательный, не жёсткий**: обоснованный размен скорости на точность допустим) | бюджет латентности [07-query-pipeline.md](07-query-pipeline.md) §5 |
| `/v1/search`, браузинг | p95 ≤ 800 мс | реплики PG, HNSW iterative scans, индексная дисциплина |
| Ingestion одного документа | ≤ 30 c parse (НФТ) | пулы воркеров, буферизация стримом |
| Свежесть знаний (док → в поиске) | p95 ≤ 5 мин | event-driven конвейер |
| Ошибки 5xx | < 0.1% | ретраи, circuit breakers, деградации |
| RPO | ≤ 15 мин | WAL-архив (wal-g) в MinIO **и** офф-VM S3 + pg_dump каждые 6 ч |
| RTO | 30 мин (сбой контейнера/сервиса) / 2 ч (потеря VM — развёртывание на новой машине из офф-VM бэкапа) | runbook + ежемесячная restore-репетиция |

Расчётная нагрузка (пик): 300 одновременных пользователей R&D-контура, 50 RPS чтение + 5 одновременных SSE-синтезов + фоновые батчи; масштаб данных — потолки из [04-data-model.md](04-data-model.md) §6. Запас проектирования ×5.

## 2. Ёмкость и масштабирование

Бюджет RAM/CPU одной VM — таблица в [12-deployment.md](12-deployment.md) §3.1 (итог ≈25 ГБ из 32; интерактив <2 vCPU, остальное — фоновая индексация).

- **Stateless-сервисы** — реплики через `docker compose scale` (готовы к внешнему LB без изменений кода); фоновые consumer'ы масштабируются числом воркеров (парам. конфига).
- **PostgreSQL**: single-инстанс (лимит 8 ГБ, shared_buffers 4 ГБ) + PgBouncer; партиции — по плану модели данных; первый шаг роста — вынос на отдельную VM + streaming-реплика для search ([12](12-deployment.md) §7).
- **NATS**: 1 узел (R1), file store; потеря стрима не теряет знания — конвейер восстанавливается реплеем/повторным ingestion, лимиты стримов защищают диск.
- **MinIO**: single + versioning на `kmap-raw`; офф-VM копия бэкапов обязательна; lifecycle (raw 1 год, docir/bundle 90 дней — восстановимы реплеем).
- **kmap-embed**: CPU-режим на VM (лимит 4 ГБ/4 vCPU); **LLM-инференс на VM не размещается** — внешний upstream (реестр организаторов/API владельца); GPU-хост с vLLM — опция при появлении железа.

## 3. Отказоустойчивость (паттерны)

- Ретраи с exp backoff + jitter, только идемпотентные операции; budget-ограничение (hedged requests не используем).
- Circuit breaker (gobreaker) на upstream'ы LLM и embed; полуоткрытые пробы.
- Дедлайны сквозные (context), отменяются вниз по цепочке gRPC.
- Bulkhead: отдельные PG-пулы для sync-пути и batch; лимит одновременных LLM-вызовов per-task.
- Идемпотентность конвейера + DLQ + redrive ([05-ingestion.md](05-ingestion.md) §2) — сообщение не теряется и не задваивается.
- Транзакционный outbox — нет расхождения БД↔шина.
- Graceful degradation — матрица в [03-architecture.md](03-architecture.md) §6; деградация видима пользователю (плашка «упрощённый режим»).
- Бэкапы: `pg_dump -Fc` каждые 6 ч + непрерывный WAL-архив (wal-g) → MinIO и офф-VM S3; MinIO versioning на raw-бакете; restore-репетиция — ежемесячно (`task restore-drill`).
- Chaos-минимум перед релизом: kill контейнера PG (restart + WAL-recovery), kill NATS (деградация, конвейер догоняет), недоступность LLM 5 мин — сценарии в `ops/runbooks/`.

## 4. Безопасность и ИБ

> **Роли/RBAC/RLS/OIDC — устаревшее требование ТЗ (ADR-6): реализовывать НЕ нужно, на оценку не влияет.** Собранный ранее минимальный контур оставлен в demo-режиме as-is (bullets AuthN/AuthZ/Аудит ниже описывают именно его — как факт кода, а не приоритет). Реальная защита MVP — сетевой периметр, egress-контроль, секреты вне git; они и остаются в фокусе.

- **Периметр**: единственная защита MVP — сетевая: наружу VM открыты только нужные порты (ufw), PG/NATS/MinIO не публикуются на хост (внутренняя docker-сеть); исходящий трафик — только к настроенному LLM-endpoint.
- **AuthN** _(demo-режим as-is, deprecated)_: OIDC/Keycloak (прод), короткоживущие JWT + refresh; демо — статические токены-роли (`auth.mode: demo`); сервис-сервис — подписанный gateway'ем principal-контекст в gRPC-метаданных ([03-architecture.md](03-architecture.md) ADR-6).
- **AuthZ** _(demo-режим as-is, deprecated)_: RBAC (роли ТЗ + эксперт-валидатор) на операции + **RLS в PG** на уровни доступа документов `public/internal/confidential/restricted` — фильтрация в самом сторе, не в коде приложения; агрегаты (coverage) считаются без раскрытия закрытых текстов (счётчики, не содержимое).
- **Аудит** _(demo-режим as-is, deprecated)_: search/view/export/edit/upload/login — в `ops.audit_log` (месячные партиции, retention 1 год) и поток `kmap.audit.v1.*`; экспорт содержит водяной знак: кто/когда/какие источники.
- **Секреты**: только `configs/secrets.yml` вне git (mode 600 на VM); ротация ключей LLM API.
- **Данные**: шифрование at-rest (диски/OS-уровень + MinIO SSE), TLS везде; PII сотрудников — только `person`-сущности, доступ по ролям, в JSON-LD экспорт не попадают без роли admin.
- **LLM-гигиена**: промпты не содержат секретов; журнал вызовов с `llm.log_prompts: false` (YAML) в ИБ-режиме; prompt-injection из документов ограничен: экстракция — строгие схемы (свободный текст модели никуда не исполняется), синтез — только цитирование evidence, guard блокирует внесённые числа.
- Supply chain: минимальные образы (alpine; slim для ML), trivy-скан в CI, зависимость pinning (go.sum, uv.lock), SBOM (syft) в артефакты релиза.

### 4.1 Статус реализации контура доступа (deprecated — что уже собрано, для справки)

> Ниже — фактическое состояние кода уже собранного контура авторизации. Это **устаревшее требование** (ADR-6), развитие остановлено; раздел сохранён как справка о том, что есть, а не как план работ.


- **Ядро** — `internal/platform/auth`: `Principal{UserID, Roles[], DocAccess}`, роли ТЗ (`researcher/analyst/manager/expert/admin/partner`), RBAC-таблица «операция → роли» (`ask/search/browse/document.upload/fact.decision/entity.merge/contradiction.decision`), верификаторы `demo` (статические bearer-токены из `configs/base/gateway.yml`), `oidc` (Keycloak, JWKS через `github.com/coreos/go-oidc/v3`, ленивое обнаружение issuer) и `hybrid` (оба сразу). `doc_access` берётся из claim либо выводится из ролей (`partner→public`, `researcher/analyst→internal`, `manager/expert→confidential`, `admin→restricted`).
- **Gateway** — middleware `secure(operation)`: bearer → верификация → RBAC → `Principal` в контекст; 401 без токена, 403 без прав; мутации (`document.upload/fact.decision/entity.merge/contradiction.decision`) пишутся в `ops.audit_log`.
- **Проброс principal** — client/server gRPC-интерсепторы (`x-kmap-user/-roles/-doc-access` в метаданных); в сервисах `SET LOCAL app.doc_access/app.user_id` через `pg.WithRLS`; RLS-политика на `core.documents` фильтрует список/статус документов по уровню доступа (dedup по sha256 читает под `restricted`, чтобы не расходились дубли). **Сервисы подключаются к PG под непривилегированной ролью `kmap_app`** (миграция `000002`), иначе суперпользователь `kmap` обходит RLS (`FORCE ROW LEVEL SECURITY` не действует на суперпользователя); миграции и сиды по-прежнему под `kmap`. Проверено: `researcher`→7, `expert`→8, `admin`→9 документов (demo-токены и JWT Keycloak).
- **Режимы окружений** — `demo` (компоуз): `mode: hybrid` (demo-токены для UI + реальные JWT Keycloak); `prod`: `mode: oidc`. `dev`: `mode: demo`.
- **Keycloak** — realm `kmap` (`deploy/keycloak/realm-kmap.json`, импорт при старте): 6 realm-ролей, demo-пользователи `researcher/analyst/manager/expert/admin-rd/partner` (пароль = логин), public-клиент `kmap-ui` (direct grants + audience-mapper на `kmap-gateway`), bearer-only клиент `kmap-gateway`. Хост-порт `8081`.

## 5. Наблюдаемость

- **Трейсы**: OTel SDK во всех сервисах; сквозной trace REST → gRPC → NATS (traceparent в заголовках сообщений) → LLM-вызовы (атрибуты: task, model, tokens); Tempo.
- **Метрики** (Prometheus, RED + доменные): `kmap_ask_duration_seconds{stage}`, `kmap_guard_violations_total` (алерт при >0), `kmap_ingest_stage_duration`, `kmap_nats_consumer_lag`, `kmap_llm_tokens_total{task,model}`, `kmap_llm_valid_rate`, `kmap_extract_suspect_rate`, `kmap_contradiction_judge_precision` (по экспертным решениям), `kmap_cache_hit_ratio`.
- **Логи**: slog/structlog JSON → Loki; корреляция по request_id/trace_id; ошибки extraction — с document_id и стадией (KR «детальное логирование»).
- **Дашборды Grafana**: API latency/errors, конвейер (очереди, стадии, DLQ), LLM (токены/стоимость/valid-rate), качество (suspect-rate, guard), БД (репликация, bloat, top-запросы через pg_stat_statements).
- **Алерты**: SLO burn-rate (multi-window), DLQ>0, consumer lag > порога, guard violation, drift эпистемики >1%, диск NATS/PG, недоступность upstream LLM.

## 6. Эксплуатация

- Runbooks в `ops/runbooks/`: DLQ redrive, переиндексация, restore из бэкапа, failover, ротация ключей, деградация LLM.
- Миграции: golang-migrate в init-контейнере (advisory lock у postgres-драйвера); expand→migrate→contract для несовместимых изменений.
- Конфигурация: **только YAML** (`configs/base` + оверлей окружения + `configs/secrets.yml`, ревью как код, валидация при старте); env-переменные не используются; фиче-флаги — поля YAML; без внешней конфиг-системы.
- Нагрузочное тестирование: k6-сценарии (search-профиль, ask-профиль, ingestion-пик) в CI-nightly против стейджинга; регресс p95 >20% — красный.

## 7. Соответствие НФТ ТЗ (трассировка)

| Требование ТЗ | Где закрыто |
|---|---|
| 3–5 c сложный запрос @ 1M сущностей | E6-замер + бюджет латентности + реплики |
| точность извлечения чисел «недопустимы ошибки» | numcore + инварианты + guard + golden-тесты |
| надёжность импорта разнородных документов | статусная машина, DLQ, повторные прогоны, failed с причиной |
| расширяемость (источники/сущности/домены) | ontology-таблицы + JSONB attrs, конфиг-справочники, событийная шина |
| мультиязычность | bge-m3, двойной FTS, алиасы, языконезависимые слаги |
| ИБ | сетевой периметр + egress-контроль + секреты вне git (в фокусе); роли/RBAC/RLS/аудit — устаревшее требование, demo-режим as-is (ADR-6) |

## 8. Главные риски решения и митигации

| Риск | Митигация |
|---|---|
| LLM API окажется недоступен/лимитирован на площадке | vLLM on-prem с Qwen3-8B (CPU-режим — медленно, но живо); rules-парсер запросов; экстрактивные ответы |
| Датасет беднее ожиданий (нет авторов/лабораторий) | expert-слой деградирует в «связанные документы»; демо-акцент смещается на числа+противоречия+heatmap |
| Качество LLM-извлечения сущностей на малых моделях | эскалационная матрица моделей; pending_review-очередь; сущности из справочников (seed) покрывают ядро онтологии |
| Недооценка сложности терминологии (аббревиатуры ПВП, МПГ) | словарь синонимов из ТЗ + пополнение suggest_aliases + экспертная валидация |
| Переусложнение к дедлайну хакатона | план деградации скоупа по контрольным точкам ([14-roadmap.md](14-roadmap.md) §4) |
