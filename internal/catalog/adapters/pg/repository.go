package pg

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maksim-mshp/nornickel-hackathon/internal/catalog/app"
	"github.com/maksim-mshp/nornickel-hackathon/internal/catalog/domain"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/auth"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/events"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/outbox"
	platformpg "github.com/maksim-mshp/nornickel-hackathon/internal/platform/pg"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const resolveByNamesSQL = `
SELECT key, id FROM (
    SELECT lower(a.alias) AS key, e.id
    FROM kg.entity_aliases a
    JOIN kg.entities e ON e.id = a.entity_id
    WHERE lower(a.alias) = ANY($1) AND e.status IN ('active','pending_review')
    UNION
    SELECT lower(canonical_name) AS key, id
    FROM kg.entities WHERE lower(canonical_name) = ANY($1) AND status IN ('active','pending_review')
) matches`

func (repository *Repository) ResolveByNames(ctx context.Context, names []string) (map[string]uuid.UUID, error) {
	if len(names) == 0 {
		return map[string]uuid.UUID{}, nil
	}
	rows, err := repository.pool.Query(ctx, resolveByNamesSQL, names)
	if err != nil {
		return nil, fmt.Errorf("resolve entities: %w", err)
	}
	defer rows.Close()

	resolved := map[string]uuid.UUID{}
	for rows.Next() {
		var key string
		var id uuid.UUID
		if err := rows.Scan(&key, &id); err != nil {
			return nil, fmt.Errorf("scan resolution: %w", err)
		}
		resolved[key] = id
	}
	return resolved, rows.Err()
}

const parameterDefsSQL = `SELECT parameter_id, plausible_min::float8, plausible_max::float8
FROM kg.parameter_defs WHERE parameter_id = ANY($1)`

func (repository *Repository) ParameterDefs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]domain.ParameterDef, error) {
	result := map[uuid.UUID]domain.ParameterDef{}
	if len(ids) == 0 {
		return result, nil
	}
	rows, err := repository.pool.Query(ctx, parameterDefsSQL, ids)
	if err != nil {
		return nil, fmt.Errorf("query parameter defs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		var min, max *float64
		if err := rows.Scan(&id, &min, &max); err != nil {
			return nil, fmt.Errorf("scan parameter def: %w", err)
		}
		result[id] = domain.ParameterDef{PlausibleMin: min, PlausibleMax: max}
	}
	return result, rows.Err()
}

const insertEntitySQL = `INSERT INTO kg.entities
(id, etype, canonical_name, canonical_name_en, slug, status, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (slug) DO UPDATE SET updated_at = now()
RETURNING id`

const insertAliasSQL = `INSERT INTO kg.entity_aliases (entity_id, alias, lang, source, status)
VALUES ($1, $2, $3, 'catalog', 'active')
ON CONFLICT DO NOTHING`

const insertChunkSQL = `INSERT INTO core.chunks
(id, document_id, version, ordinal, text, kind, page_from, page_to, char_from, char_to, lang, section_path, embedding)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13::vector)
ON CONFLICT (document_id, version, ordinal) DO UPDATE SET text = EXCLUDED.text, embedding = COALESCE(EXCLUDED.embedding, core.chunks.embedding)`

const insertFactSQL = `INSERT INTO kg.numeric_facts
(id, document_id, chunk_id, subject_id, parameter_id, relation, operator, value_raw, vmin, vmax,
 unit_orig, unit_code, vmin_si, vmax_si, conditions, condition_hash, quote, page, char_from, char_to,
 geography, doc_year, extraction_method, extractor_version, extraction_confidence, validation_status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
        $21, $22, $23, $24, $25, $26::kg.validation_status)
ON CONFLICT DO NOTHING`

const documentContextSQL = `SELECT year, geography::text FROM core.documents WHERE id = $1`
const updateDocStatusSQL = `UPDATE core.documents SET status = 'indexed', updated_at = now() WHERE id = $1`

const recomputeEdgesSQL = `
INSERT INTO kg.edges (src, dst, rel, weight, provenance)
SELECT f.subject_id, f.parameter_id, 'OPERATES_AT',
       count(*)::int,
       coalesce(jsonb_agg(DISTINCT f.document_id) FILTER (WHERE f.document_id IS NOT NULL), '[]'::jsonb)
FROM kg.numeric_facts f
WHERE (f.subject_id, f.parameter_id) IN (
        SELECT DISTINCT subject_id, parameter_id
        FROM kg.numeric_facts
        WHERE document_id = $1 AND subject_id IS NOT NULL AND parameter_id IS NOT NULL
      )
  AND f.superseded_by IS NULL
GROUP BY f.subject_id, f.parameter_id
ON CONFLICT (src, dst, rel) DO UPDATE
  SET weight = EXCLUDED.weight, provenance = EXCLUDED.provenance, updated_at = now()`
const markDocFailedSQL = `
UPDATE core.documents
SET status = 'failed',
    meta = coalesce(meta, '{}'::jsonb) || jsonb_build_object('failure_reason', $2::text),
    updated_at = now()
WHERE id = $1 AND status NOT IN ('indexed', 'archived')`

func (repository *Repository) Commit(ctx context.Context, cmd app.CommitCommand, committed events.Envelope, clusterDirty events.Envelope) error {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := platformpg.SetRLS(ctx, tx, platformpg.Principal{UserID: "system", DocAccess: auth.AccessRestricted}); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, cmd.DocumentID.String()); err != nil {
		return fmt.Errorf("acquire document lock: %w", err)
	}

	var docYear *int
	var docGeography string
	err = tx.QueryRow(ctx, documentContextSQL, cmd.DocumentID).Scan(&docYear, &docGeography)
	if err == pgx.ErrNoRows {
		return domain.ErrDocumentNotFound
	}
	if err != nil {
		return fmt.Errorf("read document context: %w", err)
	}

	remap := map[uuid.UUID]uuid.UUID{}
	for _, entity := range cmd.NewEntities {
		var actualID uuid.UUID
		err := tx.QueryRow(ctx, insertEntitySQL,
			entity.ID, entity.EType, entity.CanonicalName, nullableString(entity.CanonicalNameEN),
			entity.Slug, entity.Status, entity.CreatedBy,
		).Scan(&actualID)
		if err != nil {
			return fmt.Errorf("insert entity %q: %w", entity.CanonicalName, err)
		}
		if actualID != entity.ID {
			remap[entity.ID] = actualID
		}
		if _, err := tx.Exec(ctx, insertAliasSQL, actualID, entity.CanonicalName, "ru"); err != nil {
			return fmt.Errorf("insert alias %q: %w", entity.CanonicalName, err)
		}
	}

	remapID := func(id uuid.UUID) uuid.UUID {
		if actual, ok := remap[id]; ok {
			return actual
		}
		return id
	}

	for _, insert := range cmd.Chunks {
		chunk := insert.Chunk
		kind := chunk.Kind
		if kind == "" {
			kind = "text"
		}
		if _, err := tx.Exec(ctx, insertChunkSQL,
			insert.UUID, cmd.DocumentID, cmd.Version, chunk.Ordinal, chunk.Text, kind,
			nullableInt(chunk.PageFrom), nullableInt(chunk.PageTo),
			chunk.CharFrom, chunk.CharTo,
			nullableString(chunk.Lang), stringArray(chunk.SectionPath),
			vectorLiteral(chunk.Embedding),
		); err != nil {
			return fmt.Errorf("insert chunk %q: %w", chunk.ID, err)
		}
	}

	for _, fact := range cmd.Facts {
		geography := fact.Geography
		if geography == "" {
			geography = docGeography
		}
		status := fact.ValidationStatus
		if status == "" {
			status = domain.FactMachineExtracted
		}
		if _, err := tx.Exec(ctx, insertFactSQL,
			fact.ID, cmd.DocumentID, fact.ChunkID, remapID(fact.SubjectID), remapID(fact.ParameterID),
			fact.Relation, fact.Operator, fact.ValueRaw, fact.VMin, fact.VMax,
			nullableString(fact.UnitOrig), nullableString(fact.UnitCode), fact.VMinSI, fact.VMaxSI,
			nullableJSON(fact.Conditions), fact.ConditionHash, fact.Quote,
			nullableInt(fact.Page), nullableInt(fact.CharFrom), nullableInt(fact.CharTo),
			geography, docYear, fact.ExtractionMethod, fact.ExtractorVersion, fact.Confidence, status,
		); err != nil {
			return fmt.Errorf("insert numeric fact: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, recomputeEdgesSQL, cmd.DocumentID); err != nil {
		return fmt.Errorf("recompute edges: %w", err)
	}

	if _, err := tx.Exec(ctx, updateDocStatusSQL, cmd.DocumentID); err != nil {
		return fmt.Errorf("update document status: %w", err)
	}

	if err := outbox.Append(ctx, tx, outbox.Record{Envelope: committed, AggregateType: "facts", AggregateID: &cmd.DocumentID}); err != nil {
		return fmt.Errorf("append facts.committed: %w", err)
	}
	if err := outbox.Append(ctx, tx, outbox.Record{Envelope: clusterDirty, AggregateType: "cluster", AggregateID: &cmd.DocumentID}); err != nil {
		return fmt.Errorf("append cluster-dirty: %w", err)
	}

	return tx.Commit(ctx)
}

func (repository *Repository) UpdateFactStatus(ctx context.Context, factID string, factKind string, statusValue string, actor string, comment string) error {
	table, err := factTable(factKind)
	if err != nil {
		return err
	}
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, fmt.Sprintf(`UPDATE %s SET validation_status = $2::kg.validation_status WHERE id = $1`, table), factID, statusValue); err != nil {
		return fmt.Errorf("update fact status: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO kg.fact_history (fact_kind, fact_id, actor, action, comment)
VALUES ($1, $2, $3, 'status_changed', $4)`,
		defaultString(factKind, "numeric"), factID, defaultString(actor, "system"), comment); err != nil {
		return fmt.Errorf("write fact history: %w", err)
	}
	return tx.Commit(ctx)
}

func (repository *Repository) MergeEntities(ctx context.Context, entityID string, intoID string, actor string, comment string) error {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `UPDATE kg.entities SET status = 'merged', merged_into = $2 WHERE id = $1`, entityID, intoID); err != nil {
		return fmt.Errorf("merge entity: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE kg.entity_aliases SET entity_id = $2 WHERE entity_id = $1`, entityID, intoID); err != nil {
		return fmt.Errorf("reassign aliases: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE kg.numeric_facts SET subject_id = $2 WHERE subject_id = $1`, entityID, intoID); err != nil {
		return fmt.Errorf("reassign fact subjects: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE kg.numeric_facts SET parameter_id = $2 WHERE parameter_id = $1`, entityID, intoID); err != nil {
		return fmt.Errorf("reassign fact parameters: %w", err)
	}
	for _, side := range []string{"src", "dst"} {
		if _, err := tx.Exec(ctx, fmt.Sprintf(`
UPDATE kg.edges e SET %[1]s = $2 WHERE e.%[1]s = $1
  AND NOT EXISTS (
    SELECT 1 FROM kg.edges x
    WHERE x.rel = e.rel AND x.%[1]s = $2
      AND x.%[2]s = e.%[2]s AND x.id <> e.id
  )`, side, map[string]string{"src": "dst", "dst": "src"}[side]), entityID, intoID); err != nil {
			return fmt.Errorf("reassign edge %s: %w", side, err)
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM kg.edges WHERE src = $1 OR dst = $1 OR (src = $2 AND dst = $2)`, entityID, intoID); err != nil {
		return fmt.Errorf("cleanup merged edges: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO kg.fact_history (fact_kind, fact_id, actor, action, comment)
VALUES ('entity', $1, $2, 'merged', $3)`, entityID, defaultString(actor, "system"), comment); err != nil {
		return fmt.Errorf("write fact history: %w", err)
	}
	return tx.Commit(ctx)
}

func (repository *Repository) MarkDocumentFailed(ctx context.Context, documentID uuid.UUID, reason string) error {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := platformpg.SetRLS(ctx, tx, platformpg.Principal{UserID: "system", DocAccess: auth.AccessRestricted}); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, markDocFailedSQL, documentID, reason); err != nil {
		return fmt.Errorf("mark document failed: %w", err)
	}
	return tx.Commit(ctx)
}

func factTable(kind string) (string, error) {
	switch defaultString(kind, "numeric") {
	case "numeric":
		return "kg.numeric_facts", nil
	case "claim":
		return "kg.claims", nil
	default:
		return "", fmt.Errorf("unsupported fact kind %q", kind)
	}
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableInt(value int) any {
	if value == 0 {
		return nil
	}
	return value
}

func nullableJSON(value map[string]any) any {
	if len(value) == 0 {
		return []byte("{}")
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}
	return raw
}

func stringArray(value []string) []string {
	if value == nil {
		return []string{}
	}
	return value
}

func vectorLiteral(embedding []float32) any {
	if len(embedding) == 0 {
		return nil
	}
	parts := make([]string, len(embedding))
	for i, value := range embedding {
		parts[i] = strconv.FormatFloat(float64(value), 'g', -1, 32)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
