package pg

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maksim-mshp/nornickel-hackathon/internal/epistemic/app"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/auth"
	platformpg "github.com/maksim-mshp/nornickel-hackathon/internal/platform/pg"
)

type queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func rlsPrincipal(ctx context.Context) platformpg.Principal {
	principal, _ := auth.FromContext(ctx)
	return platformpg.Principal{UserID: principal.UserID, DocAccess: principal.DocAccess}
}

func (repo *Repo) read(ctx context.Context, fn func(queryer) error) error {
	return platformpg.WithRLS(ctx, repo.pool, rlsPrincipal(ctx), func(txCtx context.Context, tx pgx.Tx) error {
		return fn(tx)
	})
}

func (repo *Repo) Coverage(ctx context.Context, domain string) ([]app.CoverageCell, error) {
	var cells []app.CoverageCell
	err := repo.read(ctx, func(q queryer) error {
		const query = `
SELECT cc.id::text, cc.domain, coalesce(m.canonical_name, ''), coalesce(pr.canonical_name, ''),
       cc.condition_key, coalesce(cc.score, 0), cc.gap_flag, cc.gap_reasons,
       cc.docs, cc.experiments, cc.facts, cc.experts, cc.ru_docs, cc.foreign_docs,
       cc.validated_facts, coalesce(cc.last_source_year, 0), coalesce(cc.score_components, '{}'::jsonb)
FROM epi.coverage_cells cc
LEFT JOIN kg.entities m ON m.id = cc.material_id
LEFT JOIN kg.entities pr ON pr.id = cc.process_id
WHERE ($1 = '' OR cc.domain = $1)
ORDER BY cc.domain, cc.score DESC`
		rows, err := q.Query(ctx, query, domain)
		if err != nil {
			return fmt.Errorf("query coverage: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var (
				cell       app.CoverageCell
				components []byte
			)
			if err := rows.Scan(
				&cell.ID, &cell.Domain, &cell.MaterialName, &cell.ProcessName,
				&cell.ConditionKey, &cell.Score, &cell.GapFlag, &cell.GapReasons,
				&cell.Docs, &cell.Experiments, &cell.Facts, &cell.Experts, &cell.RuDocs, &cell.ForeignDocs,
				&cell.ValidatedFacts, &cell.LastSourceYear, &components,
			); err != nil {
				return fmt.Errorf("scan coverage: %w", err)
			}
			cell.ScoreComponents = decodeMap(components)
			cells = append(cells, cell)
		}
		return rows.Err()
	})
	return cells, err
}

func (repo *Repo) Contradictions(ctx context.Context, status string, entityID string) ([]app.Contradiction, error) {
	var result []app.Contradiction
	err := repo.read(ctx, func(q queryer) error {
		const query = `
SELECT ct.id::text, ct.cluster_id::text, ct.status, ct.dtype, coalesce(ct.severity, 0),
       fa.quote, fb.quote, coalesce(ct.judge_rationale, ''), coalesce(ct.confounders, '[]'::jsonb),
       sub.canonical_name, param.canonical_name
FROM epi.contradictions ct
JOIN epi.clusters c ON c.id = ct.cluster_id
JOIN kg.numeric_facts fa ON fa.id = ct.a_id
JOIN kg.numeric_facts fb ON fb.id = ct.b_id
JOIN kg.entities sub ON sub.id = c.subject_id
LEFT JOIN kg.entities param ON param.id = c.parameter_id
WHERE ($1 = '' OR ct.status = $1)
  AND ($2 = '' OR c.subject_id::text = $2 OR c.parameter_id::text = $2)
ORDER BY ct.severity DESC NULLS LAST`
		rows, err := q.Query(ctx, query, status, entityID)
		if err != nil {
			return fmt.Errorf("query contradictions: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var (
				contradiction app.Contradiction
				confounders   []byte
				paramName     *string
			)
			if err := rows.Scan(
				&contradiction.ID, &contradiction.ClusterID, &contradiction.Status, &contradiction.Dtype,
				&contradiction.Severity, &contradiction.AStatement, &contradiction.BStatement,
				&contradiction.Cause, &confounders, &contradiction.Subject, &paramName,
			); err != nil {
				return fmt.Errorf("scan contradiction: %w", err)
			}
			_ = json.Unmarshal(confounders, &contradiction.Confounders)
			if paramName != nil {
				contradiction.Parameter = *paramName
			}
			result = append(result, contradiction)
		}
		return rows.Err()
	})
	return result, err
}

func (repo *Repo) DecideContradiction(ctx context.Context, id string, status string, rationale string) (app.Contradiction, error) {
	const query = `
UPDATE epi.contradictions
SET status = $2, judge_rationale = COALESCE(NULLIF($3, ''), judge_rationale), decided_at = now()
WHERE id = $1
RETURNING id::text, status, dtype, coalesce(severity, 0)`
	var contradiction app.Contradiction
	err := repo.pool.QueryRow(ctx, query, id, status, rationale).Scan(
		&contradiction.ID, &contradiction.Status, &contradiction.Dtype, &contradiction.Severity,
	)
	if err != nil {
		return app.Contradiction{}, fmt.Errorf("update contradiction: %w", err)
	}
	return contradiction, nil
}

func (repo *Repo) RecalculateFacts(ctx context.Context, factIDs []string) ([]string, error) {
	if len(factIDs) == 0 {
		return nil, nil
	}
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := platformpg.SetRLS(ctx, tx, platformpg.Principal{UserID: "system", DocAccess: auth.AccessRestricted}); err != nil {
		return nil, err
	}

	clusterKeys, err := upsertClusters(ctx, tx, factIDs)
	if err != nil {
		return nil, err
	}
	if err := rebuildCoverage(ctx, tx, factIDs); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return clusterKeys, nil
}

func upsertClusters(ctx context.Context, tx pgx.Tx, factIDs []string) ([]string, error) {
	const query = `
WITH facts AS (
  SELECT f.id, f.subject_id, f.parameter_id, coalesce(f.condition_hash, '\x'::bytea) AS condition_hash,
         decode(md5(f.subject_id::text || ':' || f.parameter_id::text || ':' || encode(coalesce(f.condition_hash, '\x'::bytea), 'hex')), 'hex') AS ckey
  FROM kg.numeric_facts f
  WHERE f.id::text = ANY($1)
),
upserted AS (
  INSERT INTO epi.clusters (ckey, subject_id, parameter_id, kind, condition_class, size, dirty)
  SELECT ckey, subject_id, parameter_id, 'numeric', '{}'::jsonb, 0, true
  FROM facts
  ON CONFLICT (ckey) DO UPDATE SET dirty = true, updated_at = now()
  RETURNING id, ckey
),
members AS (
  INSERT INTO epi.cluster_members (cluster_id, fact_kind, fact_id)
  SELECT u.id, 'numeric', f.id
  FROM facts f
  JOIN upserted u ON u.ckey = f.ckey
  ON CONFLICT DO NOTHING
  RETURNING cluster_id
),
sizes AS (
  UPDATE epi.clusters c
  SET size = counted.size, updated_at = now()
  FROM (
    SELECT cluster_id, count(*)::int AS size
    FROM epi.cluster_members
    WHERE cluster_id IN (SELECT id FROM upserted)
    GROUP BY cluster_id
  ) counted
  WHERE c.id = counted.cluster_id
  RETURNING c.ckey
)
SELECT encode(ckey, 'hex') FROM sizes`
	rows, err := tx.Query(ctx, query, factIDs)
	if err != nil {
		return nil, fmt.Errorf("upsert clusters: %w", err)
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("scan cluster key: %w", err)
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

func rebuildCoverage(ctx context.Context, tx pgx.Tx, factIDs []string) error {
	const query = `
WITH affected AS (
  SELECT DISTINCT subject_id AS entity_id
  FROM kg.numeric_facts
  WHERE id::text = ANY($1)
),
deleted AS (
  DELETE FROM epi.coverage_cells cc
  USING affected a
  WHERE cc.process_id = a.entity_id
),
stats AS (
  SELECT f.subject_id AS process_id,
         count(DISTINCT f.document_id)::int AS docs,
         count(*)::int AS facts,
         count(*) FILTER (WHERE f.validation_status IN ('multi_source','expert_validated'))::int AS validated_facts,
         count(DISTINCT f.document_id) FILTER (WHERE f.geography = 'ru')::int AS ru_docs,
         count(DISTINCT f.document_id) FILTER (WHERE f.geography = 'foreign')::int AS foreign_docs,
         max(f.doc_year)::int AS last_source_year
  FROM kg.numeric_facts f
  WHERE f.subject_id IN (SELECT entity_id FROM affected)
    AND f.superseded_by IS NULL
  GROUP BY f.subject_id
)
INSERT INTO epi.coverage_cells
  (domain, process_id, condition_key, docs, facts, validated_facts, ru_docs, foreign_docs, last_source_year,
   score, score_components, gap_flag, gap_reasons, computed_at)
SELECT 'default', process_id, '', docs, facts, validated_facts, ru_docs, foreign_docs, last_source_year,
       LEAST(1.0, (docs::real / 5.0) + (validated_facts::real / 10.0)),
       jsonb_build_object('docs', docs, 'validated', validated_facts),
       docs < 2,
       CASE WHEN docs < 2 THEN ARRAY['insufficient_documents']::text[] ELSE '{}'::text[] END,
       now()
FROM stats`
	if _, err := tx.Exec(ctx, query, factIDs); err != nil {
		return fmt.Errorf("rebuild coverage: %w", err)
	}
	return nil
}

func decodeMap(raw []byte) map[string]float64 {
	result := map[string]float64{}
	if len(raw) == 0 {
		return result
	}
	_ = json.Unmarshal(raw, &result)
	return result
}
