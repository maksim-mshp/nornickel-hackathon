package pg

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maksim-mshp/nornickel-hackathon/internal/epistemic/app"
)

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (repo *Repo) Coverage(ctx context.Context, domain string) ([]app.CoverageCell, error) {
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
	rows, err := repo.pool.Query(ctx, query, domain)
	if err != nil {
		return nil, fmt.Errorf("query coverage: %w", err)
	}
	defer rows.Close()

	var cells []app.CoverageCell
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
			return nil, fmt.Errorf("scan coverage: %w", err)
		}
		cell.ScoreComponents = decodeMap(components)
		cells = append(cells, cell)
	}
	return cells, rows.Err()
}

func (repo *Repo) Contradictions(ctx context.Context, status string, entityID string) ([]app.Contradiction, error) {
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
	rows, err := repo.pool.Query(ctx, query, status, entityID)
	if err != nil {
		return nil, fmt.Errorf("query contradictions: %w", err)
	}
	defer rows.Close()

	var result []app.Contradiction
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
			return nil, fmt.Errorf("scan contradiction: %w", err)
		}
		_ = json.Unmarshal(confounders, &contradiction.Confounders)
		if paramName != nil {
			contradiction.Parameter = *paramName
		}
		result = append(result, contradiction)
	}
	return result, rows.Err()
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

func decodeMap(raw []byte) map[string]float64 {
	result := map[string]float64{}
	if len(raw) == 0 {
		return result
	}
	_ = json.Unmarshal(raw, &result)
	return result
}
