# 04. Модель данных и онтология

PostgreSQL 18, расширения: `pgvector 0.8`, `pg_trgm`, `btree_gin`. PK — `uuid DEFAULT uuidv7()` (нативная функция PG 18; времени-сортируемые UUID дешевле для B-tree). Все таблицы: `created_at/updated_at timestamptz NOT NULL DEFAULT now()` (в DDL ниже опущено для краткости).

Схемы: `core` (документы/чанки), `kg` (онтология, факты, граф), `epi` (эпистемика), `iam` (пользователи/доступы), `ops` (outbox, аудит, джобы), `eval` (gold-set).

## 1. Онтология: типы сущностей

Единая таблица сущностей с типом (полиморфизм через `etype` + JSONB-атрибуты; отдельные таблицы только там, где есть своя структура — люди, документы).

| etype | Описание | Примеры | Ключевые attrs (JSONB) |
|---|---|---|---|
| `material` | материалы, вещества, компоненты | никель, католит, техногенный гипс, SO₂ | formula, cas_number, class (metal/salt/solution/ore/waste) |
| `process` | технологические процессы | электроэкстракция, кучное выщелачивание, обессоливание | domain (hydro/pyro/ecology/waste), stage |
| `equipment` | оборудование и установки | ванна электроэкстракции, ПВП, обратноосмотическая установка | vendor, model |
| `facility` | производственный объект/площадка (Facility из онтологии ТЗ) | обогатительная фабрика, цех электролиза никеля, очистные сооружения | site, org_id |
| `property` | свойства и показатели (что измеряем у результата) | чистота катода, выход металла, сухой остаток | direction_good (up/down) |
| `parameter` | управляемые параметры процесса | температура, скорость потока, плотность тока, концентрация SO₄²⁻ | dimension, si_unit → связь с `kg.parameter_defs` |
| `technology` | техническое решение / метод | схема параллельной циркуляции, ионный обмен | trl |
| `experiment` | эксперимент / серия опытов | EXP-014 | catalog_row, series |
| `publication` | источник знания | статья, патент, отчёт, диссертация | doi, patent_no; 1:1 с `core.documents` |
| `person` | автор/эксперт | сотрудник, внешний автор | staff_id, position; PII-контур |
| `lab` | лаборатория/команда | лаб. гидрометаллургии | org_unit |
| `org` | организация | НИИ, вуз, компания | country |
| `geography` | географический признак | Россия, зарубежная практика, Финляндия | iso, scope (ru/foreign/global) |
| `topic` | тематический тег из таксономии | «очистка шахтных вод» | taxonomy_path |
| `economic_indicator` | экономический показатель (KR: экономика — first-class) | CAPEX, OPEX, себестоимость на тонну | currency_sensitive |
| `climate` | климатический/средовой контекст | холодный климат, Заполярье | — |

## 2. Онтология: типы связей (`kg.edges.rel`)

Суперсет RFC §9.4/§14.2 (17 типов), направление «src → dst»:

```
DESCRIBES(publication→experiment|technology)   USES_MATERIAL(experiment|process|technology→material)
USES_PROCESS(experiment|technology→process)    USES_EQUIPMENT(experiment|process→equipment)
OPERATES_AT(experiment|process→parameter)*     PRODUCES_PROPERTY(experiment→property)*
IMPROVES(process|technology|parameter→property)*   DECREASES(...)*   NO_EFFECT(...)*
APPLICABLE_FOR(technology→material|process|climate)*   NOT_APPLICABLE(...)*
SUPPORTED_BY(conclusion_claim→publication|experiment)  CONTRADICTS(claim|fact→claim|fact)†
VALIDATED_BY(fact|claim→person)                WORKED_ON(person|lab→topic|process|material)
AUTHORED(person→publication)   AFFILIATED(person→lab|org)   LOCATED_IN(publication|experiment|org→geography)
MEASURED_IN(parameter→unit)    PART_OF(topic→topic, lab→org)   RELATED_TO(entity→entity)
MENTIONED_IN(entity→chunk)     HAS_CONCLUSION(experiment|publication→claim)
INSTALLED_AT(equipment→facility)   CONDUCTED_AT(experiment→facility)   APPLIED_AT(technology→facility)
```
`*` — связи-носители фактов: материализуются строками `kg.numeric_facts` / `kg.claims` (ребро агрегирует, факт хранит значения и provenance). `†` — CONTRADICTS не пишется экстрактором напрямую; только Epistemic-контуром после judge (см. [08-epistemic.md](08-epistemic.md)).

## 3. DDL — ядро (сокращённые, но точные эскизы)

### 3.1. core — документы и чанки

```sql
CREATE TYPE core.doc_type AS ENUM ('article','report','patent','protocol','handbook','normative','dataset','web');
CREATE TYPE core.geo_scope AS ENUM ('ru','foreign','global','unknown');
CREATE TYPE core.access_level AS ENUM ('public','internal','confidential','restricted');
CREATE TYPE core.doc_status AS ENUM ('registered','parsing','parsed','extracting','extracted','committing','indexed','failed','archived');

CREATE TABLE core.documents (
  id            uuid PRIMARY KEY DEFAULT uuidv7(),
  title         text NOT NULL,
  doc_type      core.doc_type NOT NULL,
  lang          text,                          -- 'ru'|'en'|'mixed'
  year          int,
  doc_date      date,
  geography     core.geo_scope NOT NULL DEFAULT 'unknown',
  access_level  core.access_level NOT NULL DEFAULT 'internal',
  source_uri    text,                          -- откуда пришёл (путь/URL)
  sha256        bytea NOT NULL UNIQUE,         -- дедупликация содержимого
  org_id        uuid REFERENCES kg.entities(id),
  status        core.doc_status NOT NULL DEFAULT 'registered',
  current_version int NOT NULL DEFAULT 1,
  uploaded_by   uuid,
  meta          jsonb NOT NULL DEFAULT '{}'    -- authors_raw[], journal, doi, tags[]
);
ALTER TABLE core.documents ENABLE ROW LEVEL SECURITY;   -- политика: access_level ≤ doc_access принципала
-- (ADR-6; principal-контекст через SET LOCAL app.doc_access / app.user_id)

CREATE TABLE core.document_versions (
  document_id uuid REFERENCES core.documents(id),
  version     int,
  blob_uri    text NOT NULL,                   -- s3://kmap-raw/...
  docir_uri   text,                            -- s3://kmap-docir/... (после parse)
  parser_version text,
  page_count  int, table_count int,
  parsed_at   timestamptz,
  PRIMARY KEY (document_id, version)
);

CREATE TABLE core.chunks (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  document_id  uuid NOT NULL REFERENCES core.documents(id),
  version      int NOT NULL,
  ordinal      int NOT NULL,
  text         text NOT NULL,
  section_path text[],                         -- ["3 Методика","3.2 Электролиз"]
  kind         text NOT NULL DEFAULT 'text',   -- text|table_row|table|caption
  page_from    int, page_to int,
  char_from    int, char_to   int,             -- span в DocIR-тексте
  lang         text,
  token_count  int,
  embedding    vector(1024),                   -- bge-m3 dense
  tsv_ru tsvector GENERATED ALWAYS AS (to_tsvector('russian', text)) STORED,
  tsv_en tsvector GENERATED ALWAYS AS (to_tsvector('english', text)) STORED,
  meta         jsonb NOT NULL DEFAULT '{}',    -- table_id, row_index, experiment_id...
  UNIQUE (document_id, version, ordinal)
) PARTITION BY HASH (document_id);             -- 8 партиций
CREATE INDEX ON core.chunks USING hnsw (embedding vector_cosine_ops) WITH (m=16, ef_construction=64);
CREATE INDEX ON core.chunks USING gin (tsv_ru);
CREATE INDEX ON core.chunks USING gin (tsv_en);
```

### 3.2. kg — сущности, справочники, факты, рёбра

```sql
CREATE TYPE kg.entity_type AS ENUM ('material','process','equipment','property','parameter',
  'technology','experiment','publication','person','lab','org','geography','topic',
  'economic_indicator','climate','facility');
CREATE TYPE kg.entity_status AS ENUM ('active','pending_review','merged','deprecated');

CREATE TABLE kg.entities (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  etype  kg.entity_type NOT NULL,
  canonical_name    text NOT NULL,             -- русское каноническое имя
  canonical_name_en text,
  slug   text NOT NULL UNIQUE,                 -- 'process:electrowinning'
  attrs  jsonb NOT NULL DEFAULT '{}',
  embedding vector(1024),                      -- эмбеддинг имени+описания (для resolution)
  status kg.entity_status NOT NULL DEFAULT 'active',
  merged_into uuid REFERENCES kg.entities(id),
  created_by text NOT NULL DEFAULT 'system'    -- system | seed | expert:<uuid>
);
CREATE INDEX ON kg.entities (etype);
CREATE INDEX ON kg.entities USING gin (canonical_name gin_trgm_ops);

CREATE TABLE kg.entity_aliases (
  entity_id uuid REFERENCES kg.entities(id),
  alias     text NOT NULL,
  lang      text NOT NULL DEFAULT 'ru',
  source    text NOT NULL DEFAULT 'dictionary',  -- dictionary|llm|expert
  status    text NOT NULL DEFAULT 'active',      -- active|pending
  PRIMARY KEY (entity_id, alias, lang)
);
CREATE INDEX ON kg.entity_aliases USING gin (alias gin_trgm_ops);

-- Реестр единиц: RU/EN написания → каноническая единица → SI
CREATE TABLE kg.units (
  code      text PRIMARY KEY,        -- 'mg_per_l'
  names     text[] NOT NULL,         -- {'мг/л','мг/дм3','мг/дм³','mg/L','mg/dm3'}
  dimension text NOT NULL,           -- 'mass_concentration'
  si_unit   text NOT NULL,           -- 'kg/m^3'
  si_factor numeric NOT NULL,        -- 1e-3
  si_offset numeric NOT NULL DEFAULT 0   -- для °C→K
);

-- Определения параметров: измерение + диапазон правдоподобия (sanity-check)
CREATE TABLE kg.parameter_defs (
  parameter_id uuid PRIMARY KEY REFERENCES kg.entities(id),
  dimension text NOT NULL,
  si_unit   text NOT NULL,
  plausible_min numeric, plausible_max numeric,   -- напр. temperature: 173..2300 K
  notes text
);

CREATE TYPE kg.op AS ENUM ('eq','lt','lte','gt','gte','range','approx','from','to','pm'); -- pm = ±
CREATE TYPE kg.extraction_method AS ENUM ('deterministic','llm','hybrid','manual','catalog');
CREATE TYPE kg.validation_status AS ENUM ('machine_extracted','weak_evidence','multi_source',
  'expert_validated','contradicted','needs_unit_review','deprecated','rejected');

CREATE TABLE kg.numeric_facts (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  document_id uuid NOT NULL REFERENCES core.documents(id),
  chunk_id    uuid NOT NULL,
  subject_id  uuid NOT NULL REFERENCES kg.entities(id),  -- experiment|process|technology|material
  parameter_id uuid NOT NULL REFERENCES kg.entities(id),
  property_id  uuid REFERENCES kg.entities(id),          -- на какое свойство влияет (если утверждается)
  relation    text NOT NULL DEFAULT 'operates_at',       -- operates_at|constraint|produces|economic
  operator    kg.op NOT NULL,
  value_raw   text NOT NULL,                             -- '0.6–0.9 м/с' как в тексте
  vmin numeric, vmax numeric,                            -- в исходной единице
  unit_orig   text,
  unit_code   text REFERENCES kg.units(code),
  vmin_si numeric, vmax_si numeric,
  si_range numrange GENERATED ALWAYS AS
    (numrange(vmin_si, vmax_si, '[]')) STORED,           -- для && пересечений
  conditions  jsonb NOT NULL DEFAULT '{}',               -- {"temperature_c":[60,80],"climate":"cold",...}
  condition_hash bytea NOT NULL,                         -- sha256 канонизированных условий
  quote text NOT NULL,                                   -- дословная цитата
  page int, char_from int, char_to int,
  geography core.geo_scope NOT NULL DEFAULT 'unknown',
  doc_year int,
  extraction_method kg.extraction_method NOT NULL,
  extractor_version text NOT NULL,                       -- 'numcore-1.4.0'
  extraction_confidence real NOT NULL,
  validation_status kg.validation_status NOT NULL DEFAULT 'machine_extracted',
  superseded_by uuid REFERENCES kg.numeric_facts(id),    -- версионирование выводов (KR-3)
  valid_from date, valid_to date
);
CREATE INDEX ON kg.numeric_facts (parameter_id, unit_code);
CREATE INDEX ON kg.numeric_facts USING gist (si_range);
CREATE INDEX ON kg.numeric_facts USING gin (conditions jsonb_path_ops);
CREATE INDEX ON kg.numeric_facts (subject_id, parameter_id, condition_hash);

-- Качественные утверждения («повышает чистоту катода», «неприменимо в холодном климате»)
CREATE TABLE kg.claims (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  document_id uuid NOT NULL, chunk_id uuid NOT NULL,
  subject_id uuid NOT NULL REFERENCES kg.entities(id),
  predicate  text NOT NULL,        -- improves|decreases|no_effect|applicable_for|not_applicable|causes
  object_id  uuid NOT NULL REFERENCES kg.entities(id),
  polarity   smallint NOT NULL,    -- +1|-1|0
  conditions jsonb NOT NULL DEFAULT '{}', condition_hash bytea NOT NULL,
  quote text NOT NULL, page int, char_from int, char_to int,
  geography core.geo_scope, doc_year int,
  extraction_confidence real NOT NULL, extractor_version text NOT NULL,
  validation_status kg.validation_status NOT NULL DEFAULT 'machine_extracted',
  superseded_by uuid
);

-- Рёбра графа (агрегированные связи; факты ссылаются на первоисточники)
CREATE TABLE kg.edges (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  src uuid NOT NULL REFERENCES kg.entities(id),
  dst uuid NOT NULL REFERENCES kg.entities(id),
  rel text NOT NULL,
  weight real NOT NULL DEFAULT 1,           -- число подтверждающих фактов (для толщины ребра)
  confidence real,
  provenance jsonb NOT NULL DEFAULT '[]',   -- [{fact_id|claim_id|document_id}...] топ-N
  UNIQUE (src, dst, rel)
);
CREATE INDEX ON kg.edges (src, rel);
CREATE INDEX ON kg.edges (dst, rel);

-- История изменений фактов (экспертные правки, KR-3: «фиксировать автора и дату»)
CREATE TABLE kg.fact_history (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  fact_kind text NOT NULL,                  -- numeric|claim|edge|entity
  fact_id uuid NOT NULL,
  actor text NOT NULL,                      -- system|expert:<uuid>
  action text NOT NULL,                     -- created|status_changed|superseded|merged|comment
  old jsonb, new jsonb, comment text
);
```

### 3.3. epi — эпистемический слой

```sql
CREATE TABLE epi.clusters (            -- кластер сопоставимых утверждений
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  ckey bytea NOT NULL UNIQUE,          -- sha256(subject_id, parameter_id|predicate+object, condition_class)
  subject_id uuid NOT NULL, parameter_id uuid, object_id uuid,
  kind text NOT NULL,                  -- numeric|claim
  condition_class jsonb NOT NULL,      -- канонизированные полосы условий
  size int NOT NULL DEFAULT 0,
  dirty bool NOT NULL DEFAULT true     -- требует пересчёта
);
CREATE TABLE epi.cluster_members (cluster_id uuid, fact_kind text, fact_id uuid,
  PRIMARY KEY (cluster_id, fact_kind, fact_id));

CREATE TABLE epi.consensus (
  cluster_id uuid PRIMARY KEY REFERENCES epi.clusters(id),
  verdict text NOT NULL,               -- consensus|majority|split|insufficient
  agreed_range numrange,               -- взвешенный медианный диапазон (SI)
  overlap_index real,                  -- обобщённый Жаккар интервалов [0..1]
  stats jsonb NOT NULL,                -- {sources:7, ru:2, foreign:5, years:[2018,2025], weights:{...}}
  confidence real NOT NULL,
  engine_version text NOT NULL, computed_at timestamptz NOT NULL
);

CREATE TABLE epi.contradictions (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  cluster_id uuid NOT NULL REFERENCES epi.clusters(id),
  a_kind text, a_id uuid, b_kind text, b_id uuid,
  dtype text NOT NULL,                 -- range_disjoint|polarity_conflict|applicability|result_mismatch
  status text NOT NULL DEFAULT 'suspected',
    -- suspected|judge_confirmed|judge_rejected|expert_confirmed|expert_rejected|resolved
  severity real, judge_model text, judge_rationale text,
  confounders jsonb,                   -- ["температура","плотность тока"]
  conditions_delta jsonb,
  decided_at timestamptz,
  UNIQUE (a_kind, a_id, b_kind, b_id)
);

CREATE TABLE epi.coverage_cells (      -- heatmap покрытия
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  domain text NOT NULL,                -- hydro|pyro|ecology|waste
  material_id uuid, process_id uuid, condition_key text,   -- напр. climate:cold
  docs int, experiments int, facts int, experts int,
  ru_docs int, foreign_docs int, validated_facts int, last_source_year int,
  score real,                          -- 0..100, перцентильная нормировка по домену
  score_components jsonb NOT NULL,     -- прозрачность вместо одного числа
  gap_flag bool NOT NULL DEFAULT false,
  gap_reasons text[],                  -- ['no_experiments','foreign_only','stale']
  computed_at timestamptz NOT NULL,
  UNIQUE (domain, material_id, process_id, condition_key)
);

CREATE TABLE epi.expert_topics (       -- профили экспертизы (H-C)
  person_id uuid REFERENCES kg.entities(id),
  entity_id uuid REFERENCES kg.entities(id),   -- topic|process|material
  weight real NOT NULL,                -- Σ role_weight×doc_relevance×exp(-0.15×возраст_лет)
  evidence jsonb NOT NULL,             -- {documents:[...], experiments:[...], last_year:2025}
  computed_at timestamptz NOT NULL,
  PRIMARY KEY (person_id, entity_id)
);
```

### 3.4. iam / ops / eval

> Схема `iam` и RLS на `core.documents` — **устаревшее требование ТЗ (demo-режим as-is, ADR-6): не приоритет, дальше не развивается.** DDL ниже сохранён как факт кода. `ops` и `eval` к этому не относятся.

```sql
CREATE TABLE iam.users (id uuid PRIMARY KEY, oidc_sub text UNIQUE NOT NULL,
  display_name text, email text, person_id uuid, roles text[] NOT NULL DEFAULT '{researcher}',
  doc_access core.access_level NOT NULL DEFAULT 'internal');

CREATE TABLE ops.outbox (id uuid PRIMARY KEY DEFAULT uuidv7(),
  aggregate_type text, aggregate_id uuid, event_type text NOT NULL,
  payload jsonb NOT NULL, headers jsonb NOT NULL DEFAULT '{}',
  created_at timestamptz NOT NULL DEFAULT now(), published_at timestamptz);
CREATE INDEX ON ops.outbox (published_at) WHERE published_at IS NULL;

CREATE TABLE ops.audit_log (id uuid DEFAULT uuidv7(), at timestamptz NOT NULL DEFAULT now(),
  actor_id uuid, action text NOT NULL,   -- search|view_doc|export|fact_edit|login|upload|delete
  object_type text, object_id text, request_id text, ip inet, details jsonb,
  PRIMARY KEY (at, id)) PARTITION BY RANGE (at);   -- месячные партиции

CREATE TABLE ops.ingest_jobs (document_id uuid, version int, stage text, status text,
  attempt int NOT NULL DEFAULT 0, error text, started_at timestamptz, finished_at timestamptz,
  PRIMARY KEY (document_id, version, stage));

CREATE TABLE eval.gold_questions (id text PRIMARY KEY, category text NOT NULL,
  question_ru text NOT NULL, plan_expected jsonb, expected jsonb NOT NULL, sources jsonb NOT NULL);
CREATE TABLE eval.runs (id uuid PRIMARY KEY DEFAULT uuidv7(), git_sha text, config jsonb,
  started_at timestamptz, finished_at timestamptz, metrics jsonb);
```

## 4. Инварианты домена (enforce в kmap-catalog)

1. `numeric_fact`: `unit_code IS NULL → validation_status='needs_unit_review'` и факт исключён из числового поиска (KR-1: неизвестную единицу не «угадываем»).
2. `vmin_si ≤ vmax_si`; `operator='range' → оба not null`; `lte/lt → vmax_si not null`; `gte/gt/from → vmin_si not null`.
3. Значение вне `parameter_defs.plausible_[min,max]` → `pending_review`, конфиденс ↓, алерт в очередь ревью — но не потеря данных.
4. Факт никогда не удаляется: только `superseded_by` / смена `validation_status` (+строка в `fact_history`).
5. Merge сущностей: `status='merged', merged_into=X`; алиасы переносятся; рёбра и факты перепривязываются батчем; обратный указатель хранится (undo).
6. Любая мутация каталога → запись в `ops.outbox` в той же транзакции.
7. `CONTRADICTS`-рёбра создаёт только Epistemic после `judge_confirmed|expert_confirmed`.

## 5. Соответствие FAIR / JSON-LD (интероперабельность из ТЗ)

Property-graph остаётся внутренним представлением; наружу — **тонкий маппинг-слой** в JSON-LD (`GET /v1/export/jsonld`, см. [10-contracts.md](10-contracts.md)):
- `@context`: schema.org + собственный словарь `kmap:` (опубликован как статический JSON);
- `publication → schema:ScholarlyArticle | schema:Patent`, `person → schema:Person`, `lab/org → schema:Organization`, `experiment → schema:Dataset` (+ `kmap:Experiment`), числовой факт → `schema:PropertyValue {minValue, maxValue, unitCode(UN/CEFACT), kmap:operator, kmap:conditions}`, provenance → `schema:isBasedOn` + `kmap:sourceSpan`;
- идентификаторы — стабильные URI `https://<host>/kg/entity/<slug>` (Findable), доступ по HTTP+RBAC (Accessible), словарь+schema.org (Interoperable), лицензия/версия факта в выгрузке (Reusable).

## 6. Замечания по масштабу и сопровождению

- Ожидаемые объёмы (потолок НФТ): documents 10⁵, chunks 3–5·10⁶, entities 10⁶, edges 10⁷, numeric_facts 10⁷, claims 10⁶. Все горячие индексы влезают в RAM 64 ГБ реплики.
- `chunks` — hash-партиции (8); `audit_log` — range по месяцам; `numeric_facts` при росте >3·10⁷ — hash по `parameter_id` (миграция подготовлена).
- Полнотекст+вектор+GiST на одной таблице фактов сознательно: письмо редкое (batch), чтение доминирует; fillfactor 90.
- Начальные данные (`db/seeds/`): единицы (~90), parameter_defs (~40), словарь синонимов RU/EN (~200 пар из ТЗ и справочников), таксономия тегов, географии, домены. Форматы — YAML, загрузчик в kmap-catalog (`kmapctl seed`).
