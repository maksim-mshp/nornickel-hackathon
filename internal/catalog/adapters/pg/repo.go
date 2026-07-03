package pg

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maksim-mshp/nornickel-hackathon/internal/catalog/app"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/events"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/outbox"
)

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (repo *Repo) ResolveEntities(ctx context.Context, names []string) ([]app.Resolution, error) {
	const query = `
SELECT e.id::text, e.slug, e.canonical_name,
       GREATEST(
         CASE WHEN lower(a.alias) = lower($1) THEN 1.0 ELSE 0 END,
         similarity(e.canonical_name, $1)
       ) AS score
FROM kg.entities e
LEFT JOIN kg.entity_aliases a ON a.entity_id = e.id
WHERE e.status = 'active'
  AND (lower(a.alias) = lower($1) OR e.canonical_name % $1 OR lower(e.canonical_name) = lower($1))
ORDER BY score DESC
LIMIT 1`
	resolutions := make([]app.Resolution, 0, len(names))
	for _, name := range names {
		resolution := app.Resolution{Input: name, Status: "unresolved"}
		var score float64
		err := repo.pool.QueryRow(ctx, query, name).Scan(&resolution.EntityID, &resolution.Slug, &resolution.CanonicalName, &score)
		if err != nil {
			if err == pgx.ErrNoRows {
				resolutions = append(resolutions, resolution)
				continue
			}
			return nil, fmt.Errorf("resolve %q: %w", name, err)
		}
		resolution.Confidence = score
		resolution.Status = "resolved"
		if score < 0.55 {
			resolution.Status = "pending_review"
		}
		resolutions = append(resolutions, resolution)
	}
	return resolutions, nil
}

func (repo *Repo) CommitExtraction(ctx context.Context, bundle app.Bundle) (app.CommitResult, error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return app.CommitResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	result := app.CommitResult{DocumentID: bundle.DocumentID}
	entityIDBySlug := map[string]string{}

	for _, entity := range bundle.Entities {
		id, err := upsertEntity(ctx, tx, entity)
		if err != nil {
			return app.CommitResult{}, err
		}
		entityIDBySlug[entity.Slug] = id
		result.EntityIDs = append(result.EntityIDs, id)
	}

	resolveSlug := func(slug string) (string, error) {
		if id, ok := entityIDBySlug[slug]; ok {
			return id, nil
		}
		var id string
		if err := tx.QueryRow(ctx, `SELECT id::text FROM kg.entities WHERE slug = $1`, slug).Scan(&id); err != nil {
			return "", fmt.Errorf("resolve slug %q: %w", slug, err)
		}
		entityIDBySlug[slug] = id
		return id, nil
	}

	for _, chunk := range bundle.Chunks {
		if _, err := tx.Exec(ctx, `
INSERT INTO core.chunks (document_id, version, ordinal, text, kind, page_from, page_to, lang)
VALUES ($1, 1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (document_id, version, ordinal) DO NOTHING`,
			bundle.DocumentID, chunk.Ordinal, chunk.Text, defaultString(chunk.Kind, "text"),
			nullableInt(chunk.PageFrom), nullableInt(chunk.PageTo), nullableString(chunk.Lang)); err != nil {
			return app.CommitResult{}, fmt.Errorf("insert chunk: %w", err)
		}
	}

	for _, fact := range bundle.NumericFacts {
		subjectID, err := resolveSlug(fact.SubjectSlug)
		if err != nil {
			return app.CommitResult{}, err
		}
		parameterID, err := resolveSlug(fact.ParameterSlug)
		if err != nil {
			return app.CommitResult{}, err
		}
		conditions, err := json.Marshal(fact.Conditions)
		if err != nil {
			return app.CommitResult{}, fmt.Errorf("marshal conditions: %w", err)
		}
		var factID string
		validation := "machine_extracted"
		if fact.UnitCode == "" {
			validation = "needs_unit_review"
		}
		err = tx.QueryRow(ctx, `
INSERT INTO kg.numeric_facts
  (document_id, subject_id, parameter_id, operator, value_raw, vmin, vmax, unit_orig, unit_code,
   vmin_si, vmax_si, conditions, quote, page, geography, doc_year, extraction_method, extractor_version,
   extraction_confidence, validation_status)
VALUES ($1,$2,$3,$4::kg.op,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15::core.geo_scope,
        (SELECT year FROM core.documents WHERE id=$1),$16::kg.extraction_method,$17,$18,$19::kg.validation_status)
RETURNING id::text`,
			bundle.DocumentID, subjectID, parameterID, fact.Operator, fact.ValueRaw, fact.Vmin, fact.Vmax,
			nullableString(fact.UnitOrig), nullableString(fact.UnitCode), fact.VminSI, fact.VmaxSI,
			conditions, fact.Quote, nullableInt(fact.Page), defaultString(fact.Geography, "unknown"),
			defaultString(fact.ExtractionMethod, "deterministic"), defaultString(fact.ExtractorVersion, bundle.ExtractorVersion),
			fact.Confidence, validation).Scan(&factID)
		if err != nil {
			return app.CommitResult{}, fmt.Errorf("insert fact: %w", err)
		}
		result.FactIDs = append(result.FactIDs, factID)

		if _, err := tx.Exec(ctx, `
INSERT INTO kg.edges (src, dst, rel, weight)
VALUES ($1, $2, 'OPERATES_AT', 1)
ON CONFLICT (src, dst, rel) DO UPDATE SET weight = kg.edges.weight + 1`,
			subjectID, parameterID); err != nil {
			return app.CommitResult{}, fmt.Errorf("upsert edge: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, `UPDATE core.documents SET status = 'indexed', updated_at = now() WHERE id = $1`, bundle.DocumentID); err != nil {
		return app.CommitResult{}, fmt.Errorf("update document status: %w", err)
	}

	envelope, err := events.New(events.Event{
		Type:    events.FactsCommitted,
		Source:  "kmap/catalog",
		Subject: bundle.DocumentID,
		Data: map[string]any{
			"document_id": bundle.DocumentID,
			"fact_ids":    result.FactIDs,
			"entity_ids":  result.EntityIDs,
		},
	})
	if err != nil {
		return app.CommitResult{}, fmt.Errorf("build facts.committed event: %w", err)
	}
	documentUUID, err := uuid.Parse(bundle.DocumentID)
	if err != nil {
		return app.CommitResult{}, fmt.Errorf("parse document id: %w", err)
	}
	if err := outbox.Append(ctx, tx, outbox.Record{
		Envelope:      envelope,
		AggregateType: "document",
		AggregateID:   &documentUUID,
	}); err != nil {
		return app.CommitResult{}, fmt.Errorf("write outbox: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return app.CommitResult{}, fmt.Errorf("commit tx: %w", err)
	}
	return result, nil
}

func (repo *Repo) UpdateFactStatus(ctx context.Context, factID string, factKind string, statusValue string, actor string, comment string) error {
	table := "kg.numeric_facts"
	if factKind == "claim" {
		table = "kg.claims"
	}
	tx, err := repo.pool.Begin(ctx)
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

func (repo *Repo) MergeEntities(ctx context.Context, entityID string, intoID string, actor string, comment string) error {
	tx, err := repo.pool.Begin(ctx)
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
	if _, err := tx.Exec(ctx, `
INSERT INTO kg.fact_history (fact_kind, fact_id, actor, action, comment)
VALUES ('entity', $1, $2, 'merged', $3)`, entityID, defaultString(actor, "system"), comment); err != nil {
		return fmt.Errorf("write fact history: %w", err)
	}
	return tx.Commit(ctx)
}

func upsertEntity(ctx context.Context, tx pgx.Tx, entity app.BundleEntity) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `
INSERT INTO kg.entities (etype, canonical_name, canonical_name_en, slug, created_by)
VALUES ($1::kg.entity_type, $2, $3, $4, 'system')
ON CONFLICT (slug) DO UPDATE SET updated_at = now()
RETURNING id::text`,
		entity.Etype, entity.Name, nullableString(entity.NameEn), entity.Slug).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("upsert entity %q: %w", entity.Slug, err)
	}
	return id, nil
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func nullableInt(value int) *int {
	if value == 0 {
		return nil
	}
	return &value
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
