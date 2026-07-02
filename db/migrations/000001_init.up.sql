CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gin;

CREATE SCHEMA IF NOT EXISTS core;
CREATE SCHEMA IF NOT EXISTS kg;
CREATE SCHEMA IF NOT EXISTS epi;
CREATE SCHEMA IF NOT EXISTS iam;
CREATE SCHEMA IF NOT EXISTS ops;
CREATE SCHEMA IF NOT EXISTS eval;

CREATE TYPE core.doc_type AS ENUM ('article','report','patent','protocol','handbook','normative','dataset','web');
CREATE TYPE core.geo_scope AS ENUM ('ru','foreign','global','unknown');
CREATE TYPE core.access_level AS ENUM ('public','internal','confidential','restricted');
CREATE TYPE core.doc_status AS ENUM ('registered','parsing','parsed','extracting','extracted','committing','indexed','failed','archived');

CREATE TABLE core.documents (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    title text NOT NULL,
    doc_type core.doc_type NOT NULL,
    lang text,
    year int,
    doc_date date,
    geography core.geo_scope NOT NULL DEFAULT 'unknown',
    access_level core.access_level NOT NULL DEFAULT 'internal',
    source_uri text,
    sha256 bytea NOT NULL UNIQUE,
    status core.doc_status NOT NULL DEFAULT 'registered',
    current_version int NOT NULL DEFAULT 1,
    uploaded_by uuid,
    meta jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE core.documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.documents FORCE ROW LEVEL SECURITY;

CREATE POLICY documents_select ON core.documents FOR SELECT USING (
    CASE COALESCE(current_setting('app.doc_access', true), 'internal')
        WHEN 'restricted' THEN true
        WHEN 'confidential' THEN access_level <> 'restricted'
        WHEN 'internal' THEN access_level IN ('public','internal')
        ELSE access_level = 'public'
    END
);
CREATE POLICY documents_insert ON core.documents FOR INSERT WITH CHECK (true);
CREATE POLICY documents_update ON core.documents FOR UPDATE USING (true) WITH CHECK (true);

CREATE TABLE core.document_versions (
    document_id uuid NOT NULL REFERENCES core.documents(id),
    version int NOT NULL,
    blob_uri text NOT NULL,
    docir_uri text,
    parser_version text,
    page_count int,
    table_count int,
    parsed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (document_id, version)
);

CREATE TABLE core.chunks (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    document_id uuid NOT NULL REFERENCES core.documents(id),
    version int NOT NULL DEFAULT 1,
    ordinal int NOT NULL,
    text text NOT NULL,
    section_path text[] NOT NULL DEFAULT '{}',
    kind text NOT NULL DEFAULT 'text',
    page_from int,
    page_to int,
    char_from int,
    char_to int,
    lang text,
    token_count int,
    embedding vector(1024),
    tsv_ru tsvector GENERATED ALWAYS AS (to_tsvector('russian', text)) STORED,
    tsv_en tsvector GENERATED ALWAYS AS (to_tsvector('english', text)) STORED,
    meta jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (document_id, version, ordinal)
);

CREATE INDEX chunks_embedding_idx ON core.chunks USING hnsw (embedding vector_cosine_ops) WITH (m = 16, ef_construction = 64);
CREATE INDEX chunks_tsv_ru_idx ON core.chunks USING gin (tsv_ru);
CREATE INDEX chunks_tsv_en_idx ON core.chunks USING gin (tsv_en);
CREATE INDEX chunks_document_idx ON core.chunks (document_id);

CREATE TYPE kg.entity_type AS ENUM ('material','process','equipment','property','parameter','technology','experiment','publication','person','lab','org','geography','topic','economic_indicator','climate','facility');
CREATE TYPE kg.entity_status AS ENUM ('active','pending_review','merged','deprecated');

CREATE TABLE kg.entities (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    etype kg.entity_type NOT NULL,
    canonical_name text NOT NULL,
    canonical_name_en text,
    slug text NOT NULL UNIQUE,
    attrs jsonb NOT NULL DEFAULT '{}',
    embedding vector(1024),
    status kg.entity_status NOT NULL DEFAULT 'active',
    merged_into uuid REFERENCES kg.entities(id),
    created_by text NOT NULL DEFAULT 'system',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX entities_etype_idx ON kg.entities (etype);
CREATE INDEX entities_name_trgm_idx ON kg.entities USING gin (canonical_name gin_trgm_ops);
CREATE INDEX entities_embedding_idx ON kg.entities USING hnsw (embedding vector_cosine_ops) WITH (m = 16, ef_construction = 64);

CREATE TABLE kg.entity_aliases (
    entity_id uuid NOT NULL REFERENCES kg.entities(id),
    alias text NOT NULL,
    lang text NOT NULL DEFAULT 'ru',
    source text NOT NULL DEFAULT 'dictionary',
    status text NOT NULL DEFAULT 'active',
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (entity_id, alias, lang)
);

CREATE INDEX aliases_alias_trgm_idx ON kg.entity_aliases USING gin (alias gin_trgm_ops);
CREATE INDEX aliases_alias_lower_idx ON kg.entity_aliases (lower(alias));

CREATE TABLE kg.units (
    code text PRIMARY KEY,
    names text[] NOT NULL,
    dimension text NOT NULL,
    si_unit text NOT NULL,
    si_factor numeric NOT NULL,
    si_offset numeric NOT NULL DEFAULT 0
);

CREATE TABLE kg.parameter_defs (
    parameter_id uuid PRIMARY KEY REFERENCES kg.entities(id),
    dimension text NOT NULL,
    si_unit text NOT NULL,
    plausible_min numeric,
    plausible_max numeric,
    notes text
);

CREATE TYPE kg.op AS ENUM ('eq','lt','lte','gt','gte','range','approx','from','to','pm');
CREATE TYPE kg.extraction_method AS ENUM ('deterministic','llm','hybrid','manual','catalog');
CREATE TYPE kg.validation_status AS ENUM ('machine_extracted','weak_evidence','multi_source','expert_validated','contradicted','needs_unit_review','deprecated','rejected');

CREATE TABLE kg.numeric_facts (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    document_id uuid NOT NULL REFERENCES core.documents(id),
    chunk_id uuid REFERENCES core.chunks(id),
    subject_id uuid NOT NULL REFERENCES kg.entities(id),
    parameter_id uuid NOT NULL REFERENCES kg.entities(id),
    property_id uuid REFERENCES kg.entities(id),
    relation text NOT NULL DEFAULT 'operates_at',
    operator kg.op NOT NULL,
    value_raw text NOT NULL,
    vmin numeric,
    vmax numeric,
    unit_orig text,
    unit_code text REFERENCES kg.units(code),
    vmin_si numeric,
    vmax_si numeric,
    si_range numrange GENERATED ALWAYS AS (numrange(vmin_si, vmax_si, '[]')) STORED,
    conditions jsonb NOT NULL DEFAULT '{}',
    condition_hash bytea NOT NULL DEFAULT '\x'::bytea,
    quote text NOT NULL DEFAULT '',
    page int,
    char_from int,
    char_to int,
    geography core.geo_scope NOT NULL DEFAULT 'unknown',
    doc_year int,
    extraction_method kg.extraction_method NOT NULL,
    extractor_version text NOT NULL,
    extraction_confidence real NOT NULL,
    validation_status kg.validation_status NOT NULL DEFAULT 'machine_extracted',
    superseded_by uuid REFERENCES kg.numeric_facts(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX numeric_facts_parameter_idx ON kg.numeric_facts (parameter_id, unit_code);
CREATE INDEX numeric_facts_range_idx ON kg.numeric_facts USING gist (si_range);
CREATE INDEX numeric_facts_conditions_idx ON kg.numeric_facts USING gin (conditions jsonb_path_ops);
CREATE INDEX numeric_facts_cluster_idx ON kg.numeric_facts (subject_id, parameter_id, condition_hash);
CREATE INDEX numeric_facts_document_idx ON kg.numeric_facts (document_id);

CREATE TABLE kg.claims (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    document_id uuid NOT NULL REFERENCES core.documents(id),
    chunk_id uuid REFERENCES core.chunks(id),
    subject_id uuid NOT NULL REFERENCES kg.entities(id),
    predicate text NOT NULL,
    object_id uuid NOT NULL REFERENCES kg.entities(id),
    polarity smallint NOT NULL DEFAULT 0,
    conditions jsonb NOT NULL DEFAULT '{}',
    condition_hash bytea NOT NULL DEFAULT '\x'::bytea,
    quote text NOT NULL DEFAULT '',
    page int,
    char_from int,
    char_to int,
    geography core.geo_scope NOT NULL DEFAULT 'unknown',
    doc_year int,
    extraction_confidence real NOT NULL,
    extractor_version text NOT NULL,
    validation_status kg.validation_status NOT NULL DEFAULT 'machine_extracted',
    superseded_by uuid REFERENCES kg.claims(id),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX claims_subject_idx ON kg.claims (subject_id, predicate, object_id);
CREATE INDEX claims_document_idx ON kg.claims (document_id);

CREATE TABLE kg.edges (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    src uuid NOT NULL REFERENCES kg.entities(id),
    dst uuid NOT NULL REFERENCES kg.entities(id),
    rel text NOT NULL,
    weight real NOT NULL DEFAULT 1,
    confidence real,
    provenance jsonb NOT NULL DEFAULT '[]',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (src, dst, rel)
);

CREATE INDEX edges_src_idx ON kg.edges (src, rel);
CREATE INDEX edges_dst_idx ON kg.edges (dst, rel);

CREATE TABLE kg.fact_history (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    fact_kind text NOT NULL,
    fact_id uuid NOT NULL,
    actor text NOT NULL,
    action text NOT NULL,
    old jsonb,
    new jsonb,
    comment text,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX fact_history_fact_idx ON kg.fact_history (fact_kind, fact_id);

CREATE TABLE epi.clusters (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    ckey bytea NOT NULL UNIQUE,
    subject_id uuid NOT NULL,
    parameter_id uuid,
    object_id uuid,
    kind text NOT NULL,
    condition_class jsonb NOT NULL DEFAULT '{}',
    size int NOT NULL DEFAULT 0,
    dirty bool NOT NULL DEFAULT true,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX clusters_dirty_idx ON epi.clusters (dirty) WHERE dirty;

CREATE TABLE epi.cluster_members (
    cluster_id uuid NOT NULL REFERENCES epi.clusters(id),
    fact_kind text NOT NULL,
    fact_id uuid NOT NULL,
    PRIMARY KEY (cluster_id, fact_kind, fact_id)
);

CREATE TABLE epi.consensus (
    cluster_id uuid PRIMARY KEY REFERENCES epi.clusters(id),
    verdict text NOT NULL,
    agreed_range numrange,
    overlap_index real,
    stats jsonb NOT NULL DEFAULT '{}',
    confidence real NOT NULL DEFAULT 0,
    engine_version text NOT NULL,
    computed_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE epi.contradictions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    cluster_id uuid NOT NULL REFERENCES epi.clusters(id),
    a_kind text NOT NULL,
    a_id uuid NOT NULL,
    b_kind text NOT NULL,
    b_id uuid NOT NULL,
    dtype text NOT NULL,
    status text NOT NULL DEFAULT 'suspected',
    severity real,
    judge_model text,
    judge_rationale text,
    confounders jsonb,
    conditions_delta jsonb,
    decided_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (a_kind, a_id, b_kind, b_id)
);

CREATE INDEX contradictions_status_idx ON epi.contradictions (status);

CREATE TABLE epi.coverage_cells (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    domain text NOT NULL,
    material_id uuid,
    process_id uuid,
    condition_key text NOT NULL DEFAULT '',
    docs int NOT NULL DEFAULT 0,
    experiments int NOT NULL DEFAULT 0,
    facts int NOT NULL DEFAULT 0,
    experts int NOT NULL DEFAULT 0,
    ru_docs int NOT NULL DEFAULT 0,
    foreign_docs int NOT NULL DEFAULT 0,
    validated_facts int NOT NULL DEFAULT 0,
    last_source_year int,
    score real,
    score_components jsonb NOT NULL DEFAULT '{}',
    gap_flag bool NOT NULL DEFAULT false,
    gap_reasons text[] NOT NULL DEFAULT '{}',
    computed_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (domain, material_id, process_id, condition_key)
);

CREATE TABLE epi.expert_topics (
    person_id uuid NOT NULL REFERENCES kg.entities(id),
    entity_id uuid NOT NULL REFERENCES kg.entities(id),
    weight real NOT NULL,
    evidence jsonb NOT NULL DEFAULT '{}',
    computed_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (person_id, entity_id)
);

CREATE TABLE iam.users (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    oidc_sub text UNIQUE NOT NULL,
    display_name text,
    email text,
    person_id uuid,
    roles text[] NOT NULL DEFAULT '{researcher}',
    doc_access core.access_level NOT NULL DEFAULT 'internal',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE ops.outbox (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    aggregate_type text NOT NULL,
    aggregate_id uuid,
    event_type text NOT NULL,
    payload jsonb NOT NULL,
    headers jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz
);

CREATE INDEX outbox_unpublished_idx ON ops.outbox (created_at) WHERE published_at IS NULL;

CREATE TABLE ops.audit_log (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    at timestamptz NOT NULL DEFAULT now(),
    actor_id text,
    action text NOT NULL,
    object_type text,
    object_id text,
    request_id text,
    ip inet,
    details jsonb
);

CREATE INDEX audit_log_at_idx ON ops.audit_log (at);
CREATE INDEX audit_log_actor_idx ON ops.audit_log (actor_id, at);

CREATE TABLE ops.ingest_jobs (
    document_id uuid NOT NULL,
    version int NOT NULL,
    stage text NOT NULL,
    status text NOT NULL DEFAULT 'pending',
    attempt int NOT NULL DEFAULT 0,
    input_hash text,
    error text,
    started_at timestamptz,
    finished_at timestamptz,
    PRIMARY KEY (document_id, version, stage)
);

CREATE TABLE ops.llm_cache (
    key bytea PRIMARY KEY,
    task text NOT NULL,
    model text NOT NULL,
    response jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

CREATE INDEX llm_cache_expires_idx ON ops.llm_cache (expires_at);

CREATE TABLE ops.answer_cache (
    key bytea PRIMARY KEY,
    plan jsonb NOT NULL,
    answer jsonb NOT NULL,
    entity_ids uuid[] NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

CREATE INDEX answer_cache_entities_idx ON ops.answer_cache USING gin (entity_ids);

CREATE TABLE eval.gold_questions (
    id text PRIMARY KEY,
    category text NOT NULL,
    question_ru text NOT NULL,
    plan_expected jsonb,
    expected jsonb NOT NULL,
    sources jsonb NOT NULL DEFAULT '{}'
);

CREATE TABLE eval.runs (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    git_sha text,
    config jsonb,
    started_at timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    metrics jsonb
);
