# 07. Запрос → ответ: understanding, retrieval, синтез, guard

Горячий путь. Только синхронные gRPC-вызовы, без брокера. SLO: first token ≤ 1.5 c, полный ответ p95 ≤ 5 c.

## 1. QueryPlan — единое структурное представление запроса

```json
{
  "schema": "queryplan/1",
  "intent": "technology_search",
  "lang": "ru",
  "entities": {
    "materials": ["material:sulfates", "material:chlorides"],
    "processes": ["process:desalination"],
    "properties": ["property:tds"],
    "equipment": [], "topics": []
  },
  "param_constraints": [
    {"parameter": "parameter:sulfate_concentration", "op": "range",
     "vmin": 200, "vmax": 300, "unit": "mg_per_l",
     "vmin_si": 0.2, "vmax_si": 0.3, "si_unit": "kg/m^3"},
    {"parameter": "parameter:tds", "op": "lte", "vmax": 1000, "unit": "mg_per_dm3", "vmax_si": 1.0}
  ],
  "conditions": {"climate": null, "medium": null},
  "geography": "any",                    // any|ru|foreign|compare
  "time_range": {"year_from": null, "year_to": null},
  "comparison": null,                    // {"axis": "geography"|"method", "options": [...]}
  "output": {"format": "answer", "sections": ["methods","evidence","contradictions","gaps","experts"]},
  "quality": {"parser": "llm|rules", "confidence": 0.93, "unresolved_terms": []}
}
```

Intents: `technology_search | experiment_search | literature_review | expert_search | gap_analysis | contradiction_analysis | comparison | entity_lookup`.

### 1.1. Двухконтурный парсер

- **LLM-контур** (kmap-llm, задача `parse_query`, Qwen3-8B, JSON Schema strict): термины → канонические слаги через подсказку топ-30 кандидатов entity-linking (pg_trgm+вектор по строке запроса — префетч до LLM-вызова).
- **Rule-контур (fallback и кросс-чек)**: словарный матчинг сущностей (aliases, pg_trgm) + **тот же numcore** на строке запроса (числа/операторы/единицы из запроса извлекаются детерминированно — реюз грамматики [06-extraction.md](06-extraction.md)). Работает ≤50 мс, покрывает Q1/Q3/Q6-подобные формулировки.
- Кросс-чек: числовые ограничения LLM-плана **обязаны** совпасть с numcore-разбором строки запроса; расхождение → берём numcore-версию (число не может быть «интерпретировано»).

## 2. Retrieval (kmap-search)

Три канала параллельно (errgroup, дедлайн 700 мс), затем слияние:

1. **Структурный канал (факты — первичны).** SQL по `kg.numeric_facts`/`kg.claims`:
   - материал/процесс/свойство: `subject_id/parameter_id/property_id ∈ план ∪ соседи 1-hop по kg.edges (USES_*, APPLICABLE_FOR)`;
   - числовые ограничения: пересечение диапазонов в SI `si_range && numrange(:min,:max)` (GiST), семантика оператора учитывается (для `lte`-требования подходят факты с `vmax_si ≤ порога` и range-факты, целиком лежащие ниже);
   - условия: `conditions @> :required_conditions` (GIN jsonb_path_ops), география/годы — btree;
   - RLS отфильтровывает недоступные принципалу документы автоматически (ADR-6).
2. **Векторный канал.** bge-m3(запрос) → pgvector HNSW top-200 cosine, `hnsw.iterative_scan=relaxed_order` (фильтры по языку/типу/году внутри запроса — без overfiltering).
3. **Лексический канал.** `websearch_to_tsquery('russian'|'english')` по `tsv_ru||tsv_en`, top-200 c `ts_rank_cd`.

**Слияние:** RRF (k=60) векторного и лексического каналов → кандидаты-чанки; чанки, на которые ссылаются найденные структурные факты, получают буст (факт — сильнее «похожести»). → **Rerank** bge-reranker-v2-m3 (kmap-embed) топ-80 → топ-30.

**EvidencePack** (выход search):
```
facts[]      — числовые факты и claims, прошедшие структурные фильтры (с provenance)
chunks[]     — топ-30 чанков-контекстов (после rerank)
consensus[]  — epi.consensus затронутых кластеров
contradictions[] — только judge_confirmed/expert_confirmed по затронутым кластерам
gaps[]       — coverage-ячейки запрошенной комбинации с gap_flag + причины
experts[]    — топ-5 из epi.expert_topics по сущностям плана (weight, evidence)
graph        — ego-подграф: сущности плана + 1–2 hop (top-N рёбер по weight)
stats        — счётчики источников (ru/foreign, годы) для шапки ответа
```

### 2.1. Ранжирование evidence

`final_score = 0.35*match_strength + 0.25*rerank_score + 0.15*source_reliability + 0.15*validation_level + 0.10*freshness`, где `match_strength`: 1.0 — структурный матч фильтров, 0.6 — 1-hop сосед; `source_reliability`: protocol/report 1.0, article 0.9, patent 0.8, web 0.5; `validation_level`: expert_validated 1.0 … machine_extracted 0.6, contradicted 0.3; `freshness = exp(-0.1*(now_year - doc_year))`. Веса — конфиг (`configs/base/ranking.yml`), в UI у каждой строки evidence — раскрываемая разбивка компонент (прозрачность вместо магии, вывод E5).

## 3. Оркестрация ответа (kmap-answer)

```
SSE-события клиенту (по мере готовности):
  event: plan          — распознанный QueryPlan (пользователь видит, как его поняли, может поправить фильтры)
  event: evidence      — EvidencePack-срез: факты, таблица, консенсус, противоречия, пробелы, эксперты, подграф
  event: answer.delta  — токены LLM-синтеза
  event: answer.done   — финальный AnswerDoc + verification-отчёт guard'а
```

- **Intent=comparison** — детерминированная декомпозиция (урок E2): по каждой опции оси сравнения — свой структурный под-запрос; сборка сравнительной таблицы (метод × {эффективность, CAPEX/OPEX, применимость в климате, ограничения}) в коде; LLM только комментирует готовую таблицу.
- **Intent=expert_search / gap_analysis** — вообще без LLM-синтеза: структурные карточки + пояснительный шаблон.
- **literature_review** — группировка источников по методу/году/географии/уровню детализации (обзор → эксперимент → патент → нормативный документ; SQL по doc_type), консенсус/разногласия из epi, LLM пишет связки между готовыми блоками.

### 3.1. Синтез (задача `synthesize_answer`)

Вход: QueryPlan + EvidencePack (факты пронумерованы `[F1]…`, чанки `[C1]…`). Промпт-инварианты: «каждое утверждение — со ссылкой [Fi]/[Ci]; **все числа — только из фактов [Fi], копируй значение и единицу дословно**; противоречия не сглаживай — опиши оба факта и подтверждённую причину; если evidence мало (<3 источников) — скажи об этом явно» (KR «показывать пользователю недостаточность»). Выход — AnswerDoc:

```json
{"summary": "...", "confidence": 0.84,
 "methods": [{"name": "обратный осмос", "applicability": "...", "citations": ["F3","F7","C2"]}],
 "numeric_table": [{"parameter": "...", "value": "...", "unit": "...", "fact_id": "F3"}],
 "contradictions": [{"id": "...", "a": "F4", "b": "F9", "cause": "различие плотности тока", "status": "judge_confirmed"}],
 "gaps": ["нет российских данных по ионному обмену для Mg>300 мг/л"],
 "experts": [...], "citations_map": {"F3": {"doc": "...", "page": 12, "quote": "..."}}}
```

### 3.2. Numeric guard (архитектурная гарантия KR-1)

После генерации: numcore-парс текста ответа → каждое найденное число+единица переводится в SI и матчится к фактам EvidencePack (допуск 1% на округление; годы/номера ссылок исключены грамматикой). Нарушение → одна регенерация с перечнем нарушений в промпте → повторное нарушение → **экстрактивная деградация**: выдаётся шаблонный ответ из фактов и таблиц без свободного текста. Метрика `hallucinated_numbers_rate` — обязана быть 0; guard-отчёт (сколько чисел проверено) прикладывается к `answer.done` и виден в UI («все 14 чисел подтверждены источниками»).

## 4. Кэш и инвалидация

- Кэш ответов: ключ = `sha256(canonical(QueryPlan) + doc_access принципала)` (разные уровни доступа не делят кэш), PG-таблица (TTL 24 ч), отдача ≤100 мс; SSE проигрывает кэшированный AnswerDoc.
- Инвалидация: событие `facts.committed` содержит entity_ids → удаление кэш-записей, чьи планы пересекаются по сущностям (инвертированный индекс план→сущности).
- Кэш эмбеддинга запроса (LRU в kmap-embed) и кэш LLM-вызовов (kmap-llm, ключ = hash(промпт+модель), для judge/extraction) — экономия ресурсов (критерий жюри).

## 5. Бюджет латентности (p95, целевой)

| Шаг | Бюджет |
|---|---|
| gateway (auth, RLS-контекст, аудит-событие) | 20 мс |
| parse_query LLM (или rules 50 мс) | 900 мс |
| префетч entity-linking | 60 мс |
| retrieval (3 канала ∥ + RRF) | 400 мс |
| rerank top-80 | 150 мс |
| сборка EvidencePack (+consensus/gaps/experts) | 120 мс |
| **отдача `evidence`-события** | **≤1.7 c** |
| синтез first token | +0.8 c |
| синтез полный (стрим) + guard | +2–3 c |
| **итого полный ответ** | **≤5 c** |

Пользователь видит evidence раньше текста — даже при медленном LLM продукт «живой».
