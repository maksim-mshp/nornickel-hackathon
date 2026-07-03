# 10. Контракты взаимодействия: REST, gRPC, события

Три плоскости: **REST/OpenAPI 3.1** (наружу, генерация swag v2), **gRPC/protobuf** (внутри, buf-управление), **NATS/CloudEvents 1.0** (асинхронный конвейер). Единый модуль контрактов в монорепо: `contracts/{proto,openapi,events}`; buf lint + breaking check в CI — обратная совместимость обязательна (эволюция только аддитивная; смена семантики = новая версия `v2`).

## 1. Конвенции

- Идентификаторы — UUIDv7 (строки в JSON). Время — RFC 3339 UTC. Языки — BCP-47 (`ru`, `en`).
- Ошибки REST — RFC 9457 `application/problem+json`: `{type, title, status, detail, instance, request_id}`; gRPC — канонические коды + `google.rpc.ErrorInfo{reason, domain="kmap", metadata}`.
- Пагинация — курсорная: `?cursor=...&limit=...` → `{items, next_cursor}`.
- Идемпотентность мутаций — заголовок `Idempotency-Key` (хранится 24 ч).
- Версия API в пути: `/v1/...`. Deprecation — заголовок `Sunset`.
- Все REST-хендлеры аннотированы swag v2 (`// @Summary`, `// @Param`, `// @Success`, `// @Router`) → `swag init --v3.1` в CI генерирует `openapi.json`; ручное редактирование спеки запрещено.

## 2. REST API (kmap-gateway, поверхность v1)

### Поиск и ответы
```
POST /v1/ask                    SSE-стрим: events plan|evidence|answer.delta|answer.done|error
  body: {"question": str, "filters": {"geography": "any|ru|foreign|compare",
         "year_from": int?, "year_to": int?, "doc_types": [..]?, "confidence_min": float?,
         "params": [{"parameter": str, "op": str, "value": num, "unit": str}]?},
         "output": {"sections": [..]?}, "lang": "ru|en"?}
POST /v1/search                 синхронный EvidencePack без синтеза (для аналитики/интеграций)
GET  /v1/answers/{id}           сохранённый AnswerDoc (share-ссылка)
```

### Знания (браузинг)
```
GET /v1/entities?type=&q=&cursor=            поиск/список сущностей
GET /v1/entities/{id}                        карточка (+счётчики, топ-связи, эксперты)
GET /v1/entities/{id}/facts?parameter=&op=&value=&unit=   факты сущности с числовыми фильтрами
GET /v1/graph?entity_id=&depth=1..3&top_n=   ego-подграф {nodes[], edges[]} (типы, вес, confidence, contradiction-флаг)
GET /v1/experiments?material=&process=&year_from=&param=&op=&value=&unit=&cursor=
GET /v1/experts?topic=|entity_id=            топ экспертов с evidence-цепочками
GET /v1/contradictions?entity_id=&status=judge_confirmed
GET /v1/coverage?domain=&axis1=material&axis2=process    heatmap-матрица
GET /v1/gaps?material=&process=&condition=   ячейки с gap_flag + смежные рекомендации
```

### Документы и ingestion
```
POST /v1/documents                       multipart (файл + метаданные) → {document_id, status}
POST /v1/documents/batch                 манифест каталога → {accepted, duplicates, failed[]}
GET  /v1/documents/{id}                  метаданные + версии
GET  /v1/documents/{id}/status           статусы стадий конвейера
GET  /v1/documents/{id}/chunks/{chunk}   текст чанка + подсветка span (для «раскрыть цитату»)
POST /v1/documents/{id}/reindex          повторный прогон
```

### Валидация и администрирование (роли expert/admin)
```
GET  /v1/review/queue?kind=entities|facts|contradictions|orphans
POST /v1/facts/{id}/status                {status, comment} → fact_history + audit
POST /v1/entities/{id}/merge              {into_id, comment}
POST /v1/entities/{id}/status             {status: accept|reject, comment} → active|deprecated + audit
POST /v1/contradictions/{id}/decision     {decision: confirmed|rejected|resolved, comment}
GET|PUT /v1/dictionaries/synonyms|units   словари (PUT — версия+автор)
GET  /v1/audit?actor=&action=&from=&to=   выгрузка аудита
```

### Экспорт
```
POST /v1/export           {answer_id | query, format: md|csv|json|jsonld} → файл (sync ≤5 МБ)
GET  /v1/export/jsonld?entity_id=|document_id=     FAIR-выгрузка (см. 04-data-model.md §5)
```
PDF-экспорт — Phase 2 (gotenberg-сервис из готового MD).

### SSE-события `/v1/ask`
```
event: plan          data: QueryPlan (+quality.parser=llm|rules)
event: evidence      data: {facts[], chunks[], consensus[], contradictions[], gaps[], experts[], graph, stats}
event: answer.delta  data: {text}
event: answer.done   data: {answer: AnswerDoc, guard: {numbers_checked, violations: 0, degraded: false}}
event: error         data: problem+json
```

## 3. События NATS (CloudEvents 1.0, JSON)

Конверт: `{specversion:"1.0", id:<outbox.id>, source:"kmap/<service>", type:"<subject>",
time, subject:"<document_id|cluster_key>", datacontenttype:"application/json", data:{...}}`.
Заголовки NATS: `Nats-Msg-Id=<id>` (дедуп), `traceparent` (OTel).

| Subject (type) | Producer | Data (ключевое) |
|---|---|---|
| `kmap.doc.v1.registered` | ingest | document_id, version, sha256, blob_uri, declared_meta{doc_type, geography, access_level} |
| `kmap.doc.v1.parsed` | parse | document_id, version, docir_uri, lang, doc_type_detected, pages, tables |
| `kmap.doc.v1.parse-failed` | parse | document_id, reason, attempt |
| `kmap.doc.v1.extracted` | extract | document_id, version, bundle_uri, counts{chunks, numeric, entities, relations, claims}, quality{nc_suspect_rate, llm_valid_rate}, versions{numcore, prompts} |
| `kmap.facts.v1.committed` | catalog | document_id, fact_ids[≤1000, иначе uri], entity_ids, cluster_keys[], new_entities[] |
| `kmap.epistemic.v1.cluster-dirty` | catalog | cluster_keys[] |
| `kmap.epistemic.v1.updated` | epistemic | cluster_keys[], contradictions{new_confirmed[], resolved[]}, coverage_cells_changed[], expert_profiles_changed[] |
| `kmap.audit.v1.<action>` | gateway | actor, action, object, request_id, details |
| `kmap.dlq.<stage>` | dlq-риппер | исходное событие + failure{reason, attempts} |

Правила: payload ≤ 256 КБ (иначе claim-check через `*_uri`); схемы событий — JSON Schema в `contracts/events/*.schema.json`, валидация на publish в `internal/platform/events`.

## 4. gRPC (protobuf, пакет `kmap.v1`)

`contracts/proto/kmap/v1/*.proto`, генерация buf: Go (`protoc-gen-go`, `-go-grpc`) и Python (`grpcio-tools`). Ключевые сервисы (сигнатуры — конспект):

```proto
service AnswerService {
  rpc Ask(AskRequest) returns (stream AskEvent);            // plan|evidence|delta|done
}
service SearchService {
  rpc Search(QueryPlan) returns (EvidencePack);
  rpc EgoGraph(EgoGraphRequest) returns (Graph);
  rpc ListExperts(ExpertQuery) returns (ExpertList);
}
service CatalogService {
  rpc CommitExtraction(CommitRequest) returns (CommitResult);   // bundle_uri, идемпотентно
  rpc ResolveEntities(ResolveRequest) returns (ResolveResult);  // имена → канонические слаги (префетч)
  rpc MergeEntities(MergeRequest) returns (MergeResult);
  rpc UpdateFactStatus(FactStatusRequest) returns (Fact);
  rpc UpsertSeed(SeedBatch) returns (SeedResult);               // справочники
}
service LLMService {
  rpc Complete(LLMRequest) returns (LLMResponse);               // task, payload, schema_ref
  rpc CompleteStream(LLMRequest) returns (stream LLMChunk);
}
service EmbedService {
  rpc Embed(EmbedRequest) returns (EmbedResponse);              // texts[] → vectors[1024]
  rpc Rerank(RerankRequest) returns (RerankResponse);
}
service EpistemicService {
  rpc GetCoverage(CoverageQuery) returns (CoverageMatrix);
  rpc GetContradictions(ContradictionQuery) returns (ContradictionList);
  rpc DecideContradiction(Decision) returns (Contradiction);
}
service IngestService {
  rpc RegisterDocument(RegisterRequest) returns (RegisterResult);
  rpc GetStatus(DocumentRef) returns (IngestStatus);
}
```

Сквозные метаданные gRPC: `x-request-id`, `x-principal` (подписанный gateway'ем контекст: user_id, roles, doc_access → RLS-сессия в PG), `traceparent`. Дедлайны: search 2 c, embed 1 c, llm — по задаче (parse_query 3 c, synthesis 60 c stream, judge 120 c batch). Ретраи — только на `UNAVAILABLE/DEADLINE_EXCEEDED` идемпотентных методов (Search, Embed, Resolve), с экспоненциальным backoff+jitter, budget 1 повтор в горячем пути.

## 5. Общие DTO (фрагменты JSON Schema)

`QueryPlan` — [07-query-pipeline.md](07-query-pipeline.md) §1. `Fact` (в EvidencePack/REST):

```json
{"id": "0197...", "kind": "numeric", "subject": {"id": "...", "slug": "process:electrowinning", "name": "электроэкстракция"},
 "parameter": {"slug": "parameter:flow_rate", "name": "скорость потока"},
 "operator": "range", "vmin": 0.6, "vmax": 0.9, "unit": "м/с",
 "si": {"vmin": 0.6, "vmax": 0.9, "unit": "m/s"},
 "conditions": {"temperature_c": [60, 80]}, "geography": "foreign", "doc_year": 2023,
 "provenance": {"document_id": "...", "title": "Отчёт 2023", "page": 12,
   "quote": "скорость циркуляции католита составляла 0.8 м/с", "char_span": [18211, 18402]},
 "extraction": {"method": "deterministic", "version": "numcore-1.4.0", "confidence": 0.98},
 "validation_status": "multi_source",
 "score_components": {"match": 1.0, "rerank": 0.71, "source": 1.0, "validation": 0.9, "freshness": 0.82}}
```

`CoverageCell`: `{domain, material, process, condition, score, score_components{...}, counters{docs, experiments, experts, ru_docs, foreign_docs, validated}, gap_flag, gap_reasons[], neighbors_suggested[]}`.

## 6. Совместимость и версии

- proto: поля только добавляются; `reserved` для удалённых; breaking-check (buf) против main.
- события: `v1` в subject; новые поля — опциональные; consumer игнорирует неизвестные.
- REST: OpenAPI-диф в CI (oasdiff) — breaking запрещён в v1.
- DocIR/bundle: `schema`-поле; parse/extract поддерживают чтение N и N−1 версий.
