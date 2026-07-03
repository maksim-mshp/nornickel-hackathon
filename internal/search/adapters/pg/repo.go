package pg

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maksim-mshp/nornickel-hackathon/internal/search/app"
)

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (repo *Repo) ExpandEntityIDs(ctx context.Context, slugs []string) ([]string, error) {
	const query = `
WITH base AS (SELECT id FROM kg.entities WHERE slug = ANY($1)),
neigh AS (
  SELECT dst AS id FROM kg.edges WHERE src IN (SELECT id FROM base)
  UNION
  SELECT src AS id FROM kg.edges WHERE dst IN (SELECT id FROM base)
)
SELECT id::text FROM base
UNION
SELECT id::text FROM neigh`
	rows, err := repo.pool.Query(ctx, query, slugs)
	if err != nil {
		return nil, fmt.Errorf("expand entities: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (repo *Repo) Facts(ctx context.Context, entityIDs []string) ([]app.Fact, error) {
	const query = `
SELECT f.id::text, f.operator::text, f.vmin::float8, f.vmax::float8, f.unit_orig,
       f.vmin_si::float8, f.vmax_si::float8, coalesce(u.si_unit, ''),
       coalesce(f.conditions, '{}'::jsonb), f.geography::text, f.extraction_method::text,
       f.extractor_version, f.extraction_confidence, f.validation_status::text,
       s.slug, s.canonical_name, p.slug, p.canonical_name,
       d.id::text, d.title, d.doc_type::text, coalesce(f.page, 0), f.quote, coalesce(d.year, 0)
FROM kg.numeric_facts f
JOIN kg.entities s ON s.id = f.subject_id
JOIN kg.entities p ON p.id = f.parameter_id
JOIN core.documents d ON d.id = f.document_id
LEFT JOIN kg.units u ON u.code = f.unit_code
WHERE f.subject_id = ANY($1) AND f.superseded_by IS NULL
ORDER BY f.extraction_confidence DESC, f.id`
	rows, err := repo.pool.Query(ctx, query, entityIDs)
	if err != nil {
		return nil, fmt.Errorf("query facts: %w", err)
	}
	defer rows.Close()

	var facts []app.Fact
	for rows.Next() {
		var (
			fact                         app.Fact
			operator                     string
			vmin, vmax, vminSI, vmaxSI   *float64
			unitOrig, siUnit             string
			conditions                   []byte
			subjectSlug, subjectName     string
			parameterSlug, parameterName string
		)
		if err := rows.Scan(
			&fact.ID, &operator, &vmin, &vmax, &unitOrig,
			&vminSI, &vmaxSI, &siUnit,
			&conditions, &fact.Geography, &fact.ExtractionMethod,
			&fact.ExtractorVersion, &fact.Confidence, &fact.ValidationStatus,
			&subjectSlug, &subjectName, &parameterSlug, &parameterName,
			&fact.Provenance.DocumentID, &fact.Provenance.Title, &fact.Provenance.DocType,
			&fact.Provenance.Page, &fact.Provenance.Quote, &fact.Provenance.Year,
		); err != nil {
			return nil, fmt.Errorf("scan fact: %w", err)
		}

		fact.Subject = app.EntityRef{Slug: subjectSlug, Name: subjectName}
		fact.Parameter = app.EntityRef{Slug: parameterSlug, Name: parameterName}
		fact.Value = app.NumericValue{Operator: operator, Vmin: vmin, Vmax: vmax, Unit: unitOrig}
		fact.SI = app.NumericValue{Operator: operator, Vmin: vminSI, Vmax: vmaxSI, Unit: siUnit}
		fact.Conditions = decodeConditions(conditions)
		facts = append(facts, fact)
	}
	return facts, rows.Err()
}

func (repo *Repo) Consensus(ctx context.Context, entityIDs []string) ([]app.Consensus, error) {
	const query = `
SELECT p.slug, p.canonical_name, co.verdict,
       lower(co.agreed_range)::float8, upper(co.agreed_range)::float8,
       coalesce(co.overlap_index, 0), co.stats
FROM epi.consensus co
JOIN epi.clusters c ON c.id = co.cluster_id
JOIN kg.entities p ON p.id = c.parameter_id
WHERE c.subject_id = ANY($1) OR c.parameter_id = ANY($1)`
	rows, err := repo.pool.Query(ctx, query, entityIDs)
	if err != nil {
		return nil, fmt.Errorf("query consensus: %w", err)
	}
	defer rows.Close()

	var result []app.Consensus
	for rows.Next() {
		var (
			slug, name, verdict           string
			agreedMin, agreedMax, overlap float64
			stats                         []byte
		)
		if err := rows.Scan(&slug, &name, &verdict, &agreedMin, &agreedMax, &overlap, &stats); err != nil {
			return nil, fmt.Errorf("scan consensus: %w", err)
		}
		parsed := struct {
			Unit    string                `json:"unit"`
			Sources []app.ConsensusSource `json:"sources"`
		}{}
		_ = json.Unmarshal(stats, &parsed)
		result = append(result, app.Consensus{
			Parameter:    app.EntityRef{Slug: slug, Name: name},
			Unit:         parsed.Unit,
			Verdict:      verdict,
			AgreedMin:    agreedMin,
			AgreedMax:    agreedMax,
			OverlapIndex: overlap,
			Sources:      parsed.Sources,
		})
	}
	return result, rows.Err()
}

func (repo *Repo) Contradictions(ctx context.Context, entityIDs []string) ([]app.Contradiction, error) {
	const query = `
SELECT ct.id::text, ct.a_id::text, ct.b_id::text, fa.quote, fb.quote,
       coalesce(ct.judge_rationale, ''), coalesce(ct.confounders, '[]'::jsonb),
       ct.status, coalesce(ct.severity, 0)
FROM epi.contradictions ct
JOIN epi.clusters c ON c.id = ct.cluster_id
JOIN kg.numeric_facts fa ON fa.id = ct.a_id
JOIN kg.numeric_facts fb ON fb.id = ct.b_id
WHERE (c.subject_id = ANY($1) OR c.parameter_id = ANY($1))
  AND ct.status IN ('judge_confirmed','expert_confirmed')`
	rows, err := repo.pool.Query(ctx, query, entityIDs)
	if err != nil {
		return nil, fmt.Errorf("query contradictions: %w", err)
	}
	defer rows.Close()

	var result []app.Contradiction
	for rows.Next() {
		var (
			contradiction app.Contradiction
			confounders   []byte
			severity      float64
		)
		if err := rows.Scan(
			&contradiction.ID, &contradiction.AFactRef, &contradiction.BFactRef,
			&contradiction.AStatement, &contradiction.BStatement, &contradiction.Cause,
			&confounders, &contradiction.Status, &severity,
		); err != nil {
			return nil, fmt.Errorf("scan contradiction: %w", err)
		}
		_ = json.Unmarshal(confounders, &contradiction.Confounders)
		contradiction.Confidence = severity
		result = append(result, contradiction)
	}
	return result, rows.Err()
}

func (repo *Repo) Gaps(ctx context.Context, entityIDs []string) ([]app.GapCell, error) {
	const query = `
SELECT coalesce(pr.canonical_name, ''), coalesce(m.canonical_name, ''), cc.condition_key,
       coalesce(cc.score, 0), cc.gap_reasons, cc.domain
FROM epi.coverage_cells cc
LEFT JOIN kg.entities m ON m.id = cc.material_id
LEFT JOIN kg.entities pr ON pr.id = cc.process_id
WHERE cc.gap_flag AND (cc.material_id = ANY($1) OR cc.process_id = ANY($1))
ORDER BY cc.score`
	rows, err := repo.pool.Query(ctx, query, entityIDs)
	if err != nil {
		return nil, fmt.Errorf("query gaps: %w", err)
	}
	defer rows.Close()

	var result []app.GapCell
	for rows.Next() {
		var (
			gap                       app.GapCell
			processName, materialName string
			condition, domain         string
		)
		if err := rows.Scan(&processName, &materialName, &condition, &gap.Score, &gap.Reasons, &domain); err != nil {
			return nil, fmt.Errorf("scan gap: %w", err)
		}
		gap.Label = buildGapLabel(processName, materialName, condition)
		gap.Neighbors = repo.gapNeighbors(ctx, domain)
		result = append(result, gap)
	}
	return result, rows.Err()
}

func (repo *Repo) gapNeighbors(ctx context.Context, domain string) []string {
	const query = `
SELECT coalesce(pr.canonical_name, '') || ' · ' || coalesce(m.canonical_name, '')
FROM epi.coverage_cells cc
LEFT JOIN kg.entities m ON m.id = cc.material_id
LEFT JOIN kg.entities pr ON pr.id = cc.process_id
WHERE cc.domain = $1 AND NOT cc.gap_flag
ORDER BY cc.score DESC
LIMIT 2`
	rows, err := repo.pool.Query(ctx, query, domain)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var neighbors []string
	for rows.Next() {
		var label string
		if err := rows.Scan(&label); err != nil {
			return neighbors
		}
		neighbors = append(neighbors, label)
	}
	return neighbors
}

func (repo *Repo) Experts(ctx context.Context, entityIDs []string) ([]app.Expert, error) {
	const query = `
SELECT person.id::text, person.canonical_name, et.weight, et.evidence
FROM epi.expert_topics et
JOIN kg.entities person ON person.id = et.person_id
WHERE et.entity_id = ANY($1)
ORDER BY et.weight DESC
LIMIT 5`
	rows, err := repo.pool.Query(ctx, query, entityIDs)
	if err != nil {
		return nil, fmt.Errorf("query experts: %w", err)
	}
	defer rows.Close()

	var result []app.Expert
	for rows.Next() {
		var (
			expert   app.Expert
			evidence []byte
		)
		if err := rows.Scan(&expert.ID, &expert.Name, &expert.Weight, &evidence); err != nil {
			return nil, fmt.Errorf("scan expert: %w", err)
		}
		parsed := struct {
			Lab         string `json:"lab"`
			Reports     int    `json:"reports"`
			Experiments int    `json:"experiments"`
			LastYear    int    `json:"last_year"`
		}{}
		_ = json.Unmarshal(evidence, &parsed)
		expert.Lab = parsed.Lab
		expert.Reports = parsed.Reports
		expert.Experiments = parsed.Experiments
		expert.LastYear = parsed.LastYear
		result = append(result, expert)
	}
	return result, rows.Err()
}

func (repo *Repo) EgoGraph(ctx context.Context, entityIDs []string) ([]app.GraphNode, []app.GraphEdge, error) {
	const nodesQuery = `
WITH ego AS (
  SELECT unnest($1::uuid[]) AS id
),
expanded AS (
  SELECT id FROM ego
  UNION SELECT dst FROM kg.edges WHERE src IN (SELECT id FROM ego)
  UNION SELECT src FROM kg.edges WHERE dst IN (SELECT id FROM ego)
)
SELECT DISTINCT e.id::text, e.etype::text, e.canonical_name
FROM kg.entities e JOIN expanded x ON x.id = e.id`
	nodeRows, err := repo.pool.Query(ctx, nodesQuery, entityIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("query graph nodes: %w", err)
	}
	defer nodeRows.Close()
	nodeSet := map[string]bool{}
	var nodes []app.GraphNode
	for nodeRows.Next() {
		var node app.GraphNode
		if err := nodeRows.Scan(&node.ID, &node.Type, &node.Label); err != nil {
			return nil, nil, err
		}
		nodeSet[node.ID] = true
		nodes = append(nodes, node)
	}
	if err := nodeRows.Err(); err != nil {
		return nil, nil, err
	}

	const edgesQuery = `
SELECT ed.id::text, ed.src::text, ed.dst::text, ed.rel, ed.weight, coalesce(ed.confidence, 0)
FROM kg.edges ed
WHERE ed.src = ANY($1) OR ed.dst = ANY($1)`
	edgeRows, err := repo.pool.Query(ctx, edgesQuery, keys(nodeSet))
	if err != nil {
		return nil, nil, fmt.Errorf("query graph edges: %w", err)
	}
	defer edgeRows.Close()
	var edges []app.GraphEdge
	for edgeRows.Next() {
		var edge app.GraphEdge
		if err := edgeRows.Scan(&edge.ID, &edge.Src, &edge.Dst, &edge.Rel, &edge.Weight, &edge.Confidence); err != nil {
			return nil, nil, err
		}
		if !nodeSet[edge.Src] || !nodeSet[edge.Dst] {
			continue
		}
		edge.Contradiction = edge.Rel == "CONTRADICTS"
		edges = append(edges, edge)
	}
	return nodes, edges, edgeRows.Err()
}

func decodeConditions(raw []byte) map[string]string {
	result := map[string]string{}
	if len(raw) == 0 {
		return result
	}
	_ = json.Unmarshal(raw, &result)
	return result
}

func buildGapLabel(processName string, materialName string, condition string) string {
	label := processName
	if materialName != "" {
		label += " · " + materialName
	}
	if condition != "" {
		label += " · " + condition
	}
	return label
}

func keys(set map[string]bool) []string {
	result := make([]string, 0, len(set))
	for key := range set {
		result = append(result, key)
	}
	return result
}
