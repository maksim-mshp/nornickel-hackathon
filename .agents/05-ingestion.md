# 05. Конвейер ingestion

Асинхронный event-driven конвейер: NATS JetStream ≥2.12, CloudEvents 1.0, claim-check для тяжёлых payload'ов (MinIO). Полные схемы событий — [10-contracts.md](10-contracts.md) §3.

## 1. Схема потока

```
(1) REGISTER   kmap-ingest    файл → MinIO kmap-raw/, sha256, реестр, outbox
(2) PARSE      kmap-parse     Docling → DocIR → MinIO kmap-docir/
(3) EXTRACT    kmap-extract   чанкинг → эмбеддинги → numeric core → LLM extraction
                              → ExtractionBundle → MinIO kmap-bundles/
(4) COMMIT     kmap-catalog   entity resolution → валидация → транзакционный коммит
(5) EPISTEMIC  kmap-epistemic dirty-кластеры → консенсус → противоречия(judge) → coverage → эксперты
(6) POSTCOMMIT инвалидация кэша ответов; notify (Phase 2); метрики
```

Статусная машина документа (`core.documents.status`): `registered → parsing → parsed → extracting → extracted → committing → indexed`; из любого этапа → `failed` (с `ops.ingest_jobs.error`), повторный запуск — идемпотентный re-drive с той же стадии.

## 2. JetStream-конфигурация

| Stream | Subjects | Retention | Storage | Replicas | Назначение |
|---|---|---|---|---|---|
| `DOCS` | `kmap.doc.v1.>` | limits, 30d / 10 ГБ | file | 3 | этапы конвейера |
| `FACTS` | `kmap.facts.v1.>` | limits, 30d | file | 3 | коммиты фактов |
| `EPI` | `kmap.epistemic.v1.>` | limits, 7d | file | 3 | dirty/updated |
| `AUDIT` | `kmap.audit.v1.>` | limits, 90d | file | 3 | аудит-события (дублируются в PG) |
| `DLQ` | `kmap.dlq.>` | limits, 14d | file | 3 | мёртвые сообщения |

Durable pull-консьюмеры (по одному на сервис-этап): `max_deliver=5`, `ack_wait=120s` (parse/extract — 15m), backoff `[10s,1m,5m,15m,1h]`. После исчерпания `max_deliver` advisory `$JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES` перекладывает сообщение в `kmap.dlq.<stage>` (сервис-риппер) с сохранением исходных заголовков; повторный прогон DLQ — командой `kmapctl dlq redrive --stage=...`.

**Идемпотентность (сквозная):**
- публикация: `Nats-Msg-Id = <outbox.id>` → JetStream-дедуп (окно 10 мин) + консьюмерская проверка;
- обработка: каждая стадия начинает с UPSERT в `ops.ingest_jobs (document_id, version, stage)`; если `status='done'` и `input_hash` совпал — ACK без работы;
- артефакты в MinIO пишутся по детерминированным ключам `s3://kmap-docir/<doc_id>/<version>/docir.json` — повторная запись безвредна;
- коммит в catalog — в одной транзакции с natural-ключами (`ON CONFLICT DO NOTHING` по `(document_id, extractor_version, span)`).

## 3. Этап 1: Register (kmap-ingest, Go)

Входы: (а) `POST /v1/documents` multipart; (б) `POST /v1/documents/batch` — манифест каталога файлов (путь смонтирован/S3); (в) URL веб-ресурса (fetcher с белым списком доменов).

Шаги: потоковая запись в MinIO + подсчёт sha256 → дедуп по `core.documents.sha256` (дубль → 200 с существующим id, версия не создаётся) → извлечение декларируемых метаданных из формы/манифеста (title, doc_type, year, geography, access_level, теги) → `core.documents`+`core.document_versions` → outbox `kmap.doc.v1.registered`.

Отдельный вход для **структурированных источников** (не проходят parse/LLM):
- каталог экспериментов CSV/XLSX → маппинг колонок (конфиг `configs/base/ingest/experiments-mapping.yml`) → строки уходят сразу в catalog как эксперименты + numeric-факты с `extraction_method='catalog'`, `confidence=0.99`; текстовое представление строки (шаблон из RFC §12.2) идёт в chunks для семантического поиска;
- справочники материалов/оборудования/единиц, перечень сотрудников/лабораторий, таксономия тегов → `kmapctl seed` → kmap-catalog (сущности со `created_by='seed'`).

## 4. Этап 2: Parse (kmap-parse, Python + Docling)

- Docling 2.x: PDF/DOCX/HTML/PPTX/XLSX → DoclingDocument; конвертация в наш **DocIR** (стабильный внутренний формат, версия схемы `docir/1`):

```json
{
  "schema": "docir/1", "document_id": "...", "version": 1,
  "lang": "ru", "lang_confidence": 0.98,
  "doc_type_detected": "report",
  "meta": {"title": "...", "authors_raw": ["Иванов И.И."], "year_detected": 2023},
  "blocks": [
    {"id": "b41", "kind": "paragraph", "page": 12, "section_path": ["3 Методика"],
     "text": "Скорость циркуляции католита составляла 0.8 м/с...",
     "char_from": 18211, "char_to": 18402},
    {"id": "t3", "kind": "table", "page": 14, "caption": "Таблица 2 — Режимы",
     "cells": [...], "rows_norm": [{"параметр": "температура", "значение": "60–80", "ед.": "°C"}]}
  ],
  "full_text_uri": "s3://kmap-docir/<id>/1/fulltext.txt"
}
```

- определение языка (fasttext lid) и типа документа (правила по структуре + zero-shot фолбэк);
- OCR выключен (текстовый слой гарантирован организаторами), включается в YAML-конфиге (`parse.ocr: auto`, Docling OCR-профиль) — деградация скорости, не архитектуры;
- сохранение исходной разметки span'ов — обязательное (provenance до символа);
- лимиты: файл ≤ 200 МБ, страницы ≤ 2000; таймаут парсинга 10 мин; битые файлы → `failed` с человекочитаемой причиной (KR «Надёжность»).

## 5. Этап 3: Extract (kmap-extract, Python)

Один консьюмер, четыре подэтапа над DocIR (детали алгоритмов — [06-extraction.md](06-extraction.md)):

1. **Чанкинг**: секционно-осознанный, 800–1200 токенов, overlap 120, таблицы — по строкам (`kind='table_row'`, meta с table_id/row_index); заголовки секций дублируются в текст чанка.
2. **Эмбеддинги**: батчи по 64 → kmap-embed (gRPC), dense 1024d (sparse-веса bge-m3 сохраняются в bundle для будущего гибрида, в MVP не используются).
3. **Numeric core** (детерминированный): грамматика чисел/операторов/единиц → кандидаты числовых фактов с точными span'ами.
4. **LLM extraction** (через kmap-llm): сущности/связи/выводы/привязка чисел к сущностям (JSON Schema, temperature 0.1); модель по умолчанию Qwen3-8B, эскалация до 30B-A3B при низком structural-valid rate по документу.

Выход — **ExtractionBundle** (`s3://kmap-bundles/<doc>/<v>/bundle.json`): `{chunks[], embeddings_uri, numeric_candidates[], entities[], relations[], claims[], quality: {nc_count, llm_valid_rate, suspects}}` → событие `kmap.doc.v1.extracted`.

## 6. Этап 4: Commit (kmap-catalog, Go)

Единственная точка записи знаний (core domain). На bundle:

1. **Entity resolution** (детерминированный каскад, LLM не участвует):
   a. точное совпадение алиаса (lower, lang-aware);
   b. pg_trgm similarity ≥ 0.55 по alias/canonical_name того же etype;
   c. cosine ≥ 0.86 по эмбеддингу имени (pgvector по kg.entities.embedding);
   d. иначе — новая сущность `status='pending_review'` (очередь ревью в админке).
   Спорные (b/c в «серой зоне» 0.45–0.55 / 0.80–0.86) — `pending_review` + текущая привязка к ближайшему кандидату c пометкой.
2. **Валидация фактов** — инварианты из [04-data-model.md](04-data-model.md) §4 (единицы, диапазоны правдоподобия, операторы).
3. **Транзакционный коммит**: chunks (+embedding) → entities/aliases → numeric_facts/claims → пересчёт агрегатов kg.edges (weight, provenance top-5) → document.status='indexed' → outbox: `kmap.facts.v1.committed {document_id, fact_ids, entity_ids, cluster_keys[]}` (+ `kmap.epistemic.v1.cluster-dirty`).
4. **Пересчёт validation_status**: факт, чьё утверждение подтверждено ≥2 независимыми документами того же кластера → `multi_source` (батч-джоб).

## 7. Этап 5: Epistemic (kmap-epistemic, Go)

По `cluster-dirty`: пересчёт только затронутых `epi.clusters` (ckey из события): membership → consensus → contradiction candidates → LLM-judge (батчами, вне горячего пути) → coverage-ячейки затронутых пар → expert_topics затронутых персон. Ночной полный пересчёт (cron 02:00) сверяет инкременты (drift-детектор: расхождение >1% — алерт). Алгоритмы — [08-epistemic.md](08-epistemic.md).

## 8. Переиндексация и версии экстракторов

- Новый `extractor_version` (улучшили грамматику/промпт) → `kmapctl reindex --from-stage=extract --filter=...`: republish `doc.parsed` для выбранных документов; факты новой версии коммитятся рядом, старые помечаются `superseded_by` — история сохраняется (KR-3, «не удалять старые версии»).
- Bump `docir` схемы → replay со стадии parse.
- Replay безопасен благодаря идемпотентности стадий (см. §2).

## 9. Производительность и масштабирование конвейера

| Стадия | Цель | Масштабирование |
|---|---|---|
| register | ≤ 1 с/файл | реплики stateless |
| parse | ≤ 30 с/документ (НФТ) | воркеры по CPU; тяжёлые PDF — GPU-профиль опционально |
| extract | ≤ 60 с/документ (числа ≤ 5 с, LLM — остальное) | параллелизм по чанкам; лимит одновременных LLM-вызовов из kmap-llm |
| commit | ≤ 3 с/bundle | батчевые UPSERT'ы, COPY для chunks |
| epistemic | ≤ 10 с инкремент | пул воркеров по cluster_key (шардирование по hash) |

Пропускная способность цели: 50 док/мин пиково (буферизуется стримом DOCS), 10⁴ документов — первичная индексация ≤ 8 ч на 4 воркерах extract при быстром LLM-upstream'е (реестр организаторов / vLLM на GPU-хосте; на одной 8 vCPU VM лимитирует CPU embed/parse — корпус хакатона в сотни–тысячи документов индексируется за часы, приемлемо).
