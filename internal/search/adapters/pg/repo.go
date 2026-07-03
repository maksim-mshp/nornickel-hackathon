package pg

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/auth"
	platformpg "github.com/maksim-mshp/nornickel-hackathon/internal/platform/pg"
	"github.com/maksim-mshp/nornickel-hackathon/internal/search/app"
	"google.golang.org/protobuf/types/known/structpb"
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

func (repo *Repo) ExpandEntityIDs(ctx context.Context, slugs []string) ([]string, error) {
	var ids []string
	err := repo.read(ctx, func(q queryer) error {
		result, err := expandEntityIDs(ctx, q, slugs)
		ids = result
		return err
	})
	return ids, err
}

func expandEntityIDs(ctx context.Context, q queryer, slugs []string) ([]string, error) {
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
	rows, err := q.Query(ctx, query, slugs)
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

func (repo *Repo) Facts(ctx context.Context, entityIDs []string, filter app.FactFilter) ([]app.Fact, error) {
	var facts []app.Fact
	err := repo.read(ctx, func(q queryer) error {
		result, err := queryFacts(ctx, q, entityIDs, filter)
		facts = result
		return err
	})
	return facts, err
}

func queryFacts(ctx context.Context, q queryer, entityIDs []string, filter app.FactFilter) ([]app.Fact, error) {
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
WHERE (f.subject_id = ANY($1) OR f.parameter_id = ANY($1)) AND f.superseded_by IS NULL
  AND f.validation_status NOT IN ('needs_unit_review', 'rejected', 'deprecated')
  AND ($2 = '' OR f.geography::text = $2)
  AND (
    $3::text[] IS NULL
    OR cardinality($3::text[]) = 0
    OR p.slug <> ALL($3::text[])
    OR EXISTS (
      SELECT 1 FROM unnest($3::text[], $4::float8[], $5::float8[]) AS c(slug, lo, hi)
      WHERE c.slug = p.slug AND f.si_range && numrange(c.lo::numeric, c.hi::numeric, '[]')
    )
  )
ORDER BY f.extraction_confidence DESC, f.id`
	rows, err := q.Query(ctx, query, entityIDs, filter.Geography, filter.ParamSlugs, filter.RangeLo, filter.RangeHi)
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
	var result []app.Consensus
	err := repo.read(ctx, func(q queryer) error {
		items, err := queryConsensus(ctx, q, entityIDs)
		result = items
		return err
	})
	return result, err
}

func queryConsensus(ctx context.Context, q queryer, entityIDs []string) ([]app.Consensus, error) {
	const query = `
SELECT p.slug, p.canonical_name, co.verdict,
       lower(co.agreed_range)::float8, upper(co.agreed_range)::float8,
       coalesce(co.overlap_index, 0), co.stats
FROM epi.consensus co
JOIN epi.clusters c ON c.id = co.cluster_id
JOIN kg.entities p ON p.id = c.parameter_id
WHERE c.subject_id = ANY($1) OR c.parameter_id = ANY($1)`
	rows, err := q.Query(ctx, query, entityIDs)
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
	var result []app.Contradiction
	err := repo.read(ctx, func(q queryer) error {
		items, err := queryContradictions(ctx, q, entityIDs)
		result = items
		return err
	})
	return result, err
}

func queryContradictions(ctx context.Context, q queryer, entityIDs []string) ([]app.Contradiction, error) {
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
	rows, err := q.Query(ctx, query, entityIDs)
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
	var result []app.GapCell
	err := repo.read(ctx, func(q queryer) error {
		items, err := queryGaps(ctx, q, entityIDs)
		result = items
		return err
	})
	return result, err
}

func queryGaps(ctx context.Context, q queryer, entityIDs []string) ([]app.GapCell, error) {
	const query = `
SELECT coalesce(pr.canonical_name, ''), coalesce(m.canonical_name, ''), cc.condition_key,
       coalesce(cc.score, 0), cc.gap_reasons, cc.domain
FROM epi.coverage_cells cc
LEFT JOIN kg.entities m ON m.id = cc.material_id
LEFT JOIN kg.entities pr ON pr.id = cc.process_id
WHERE cc.gap_flag AND (cc.material_id = ANY($1) OR cc.process_id = ANY($1))
ORDER BY cc.score`
	rows, err := q.Query(ctx, query, entityIDs)
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
		if gap.Reasons == nil {
			gap.Reasons = []string{}
		}
		gap.Neighbors = gapNeighbors(ctx, q, domain)
		result = append(result, gap)
	}
	return result, rows.Err()
}

func gapNeighbors(ctx context.Context, q queryer, domain string) []string {
	const query = `
SELECT coalesce(pr.canonical_name, '') || ' · ' || coalesce(m.canonical_name, '')
FROM epi.coverage_cells cc
LEFT JOIN kg.entities m ON m.id = cc.material_id
LEFT JOIN kg.entities pr ON pr.id = cc.process_id
WHERE cc.domain = $1 AND NOT cc.gap_flag
ORDER BY cc.score DESC
LIMIT 2`
	neighbors := []string{}
	rows, err := q.Query(ctx, query, domain)
	if err != nil {
		return neighbors
	}
	defer rows.Close()
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
	var result []app.Expert
	err := repo.read(ctx, func(q queryer) error {
		items, err := queryExperts(ctx, q, entityIDs)
		result = items
		return err
	})
	return result, err
}

func queryExperts(ctx context.Context, q queryer, entityIDs []string) ([]app.Expert, error) {
	const query = `
SELECT person.id::text, person.canonical_name, et.weight, et.evidence
FROM epi.expert_topics et
JOIN kg.entities person ON person.id = et.person_id
WHERE et.entity_id = ANY($1)
ORDER BY et.weight DESC
LIMIT 5`
	rows, err := q.Query(ctx, query, entityIDs)
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
	var (
		nodes []app.GraphNode
		edges []app.GraphEdge
	)
	err := repo.read(ctx, func(q queryer) error {
		n, e, err := egoGraph(ctx, q, entityIDs)
		nodes, edges = n, e
		return err
	})
	return nodes, edges, err
}

func egoGraph(ctx context.Context, q queryer, entityIDs []string) ([]app.GraphNode, []app.GraphEdge, error) {
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
	nodeRows, err := q.Query(ctx, nodesQuery, entityIDs)
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
	edgeRows, err := q.Query(ctx, edgesQuery, keys(nodeSet))
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

func (repo *Repo) ListEntities(ctx context.Context, entityType string, query string, limit uint32) ([]*kmapv1.EntitySummary, error) {
	var result []*kmapv1.EntitySummary
	err := repo.read(ctx, func(q queryer) error {
		items, err := listEntities(ctx, q, entityType, query, limit)
		result = items
		return err
	})
	return result, err
}

func listEntities(ctx context.Context, q queryer, entityType string, query string, limit uint32) ([]*kmapv1.EntitySummary, error) {
	const sql = `
SELECT e.id::text, e.slug, e.canonical_name, coalesce(e.canonical_name_en, ''), e.etype::text,
       count(DISTINCT f.id)::int, count(DISTINCT ed.id)::int
FROM kg.entities e
LEFT JOIN kg.numeric_facts f ON f.subject_id = e.id OR f.parameter_id = e.id
LEFT JOIN kg.edges ed ON ed.src = e.id OR ed.dst = e.id
WHERE e.status = 'active'
  AND ($1 = '' OR e.etype::text = $1)
  AND ($2 = '' OR e.slug ILIKE '%' || $2 || '%' OR e.canonical_name ILIKE '%' || $2 || '%')
GROUP BY e.id, e.slug, e.canonical_name, e.canonical_name_en, e.etype
ORDER BY count(DISTINCT f.id) DESC, e.canonical_name
LIMIT $3`
	rows, err := q.Query(ctx, sql, entityType, query, int(limit))
	if err != nil {
		return nil, fmt.Errorf("query entities: %w", err)
	}
	defer rows.Close()

	var result []*kmapv1.EntitySummary
	for rows.Next() {
		item := &kmapv1.EntitySummary{}
		if err := rows.Scan(&item.Id, &item.Slug, &item.Name, &item.NameEn, &item.Etype, &item.Facts, &item.Relations); err != nil {
			return nil, fmt.Errorf("scan entity: %w", err)
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (repo *Repo) GetEntity(ctx context.Context, entityID string) (*kmapv1.EntityCard, error) {
	var card *kmapv1.EntityCard
	err := repo.read(ctx, func(q queryer) error {
		item, err := getEntity(ctx, q, entityID)
		card = item
		return err
	})
	return card, err
}

func getEntity(ctx context.Context, q queryer, entityID string) (*kmapv1.EntityCard, error) {
	const sql = `
SELECT e.id::text, e.slug, e.canonical_name, coalesce(e.canonical_name_en, ''), e.etype::text,
       count(DISTINCT f.id)::int, count(DISTINCT ed.id)::int
FROM kg.entities e
LEFT JOIN kg.numeric_facts f ON f.subject_id = e.id OR f.parameter_id = e.id
LEFT JOIN kg.edges ed ON ed.src = e.id OR ed.dst = e.id
WHERE e.id::text = $1 OR e.slug = $1
GROUP BY e.id, e.slug, e.canonical_name, e.canonical_name_en, e.etype`
	item := &kmapv1.EntityCard{}
	var facts, relations uint32
	if err := q.QueryRow(ctx, sql, entityID).Scan(&item.Id, &item.Slug, &item.NameRu, &item.NameEn, &item.Type, &facts, &relations); err != nil {
		return nil, fmt.Errorf("query entity: %w", err)
	}
	counters, err := structpb.NewStruct(map[string]any{"facts": float64(facts), "relations": float64(relations)})
	if err != nil {
		return nil, fmt.Errorf("build counters: %w", err)
	}
	item.Counters = counters
	item.Synonyms = entityAliases(ctx, q, item.Id)
	_, edges, err := egoGraph(ctx, q, []string{item.Id})
	if err != nil {
		return nil, err
	}
	item.Relations = make([]*kmapv1.GraphEdge, 0, len(edges))
	for _, edge := range edges {
		item.Relations = append(item.Relations, &kmapv1.GraphEdge{
			Id: edge.ID, Src: edge.Src, Dst: edge.Dst, Rel: edge.Rel,
			Weight: edge.Weight, Confidence: edge.Confidence, Contradiction: edge.Contradiction,
		})
	}
	return item, nil
}

func (repo *Repo) ListExperiments(ctx context.Context, req *kmapv1.ListExperimentsRequest) ([]*kmapv1.ExperimentSummary, error) {
	var result []*kmapv1.ExperimentSummary
	err := repo.read(ctx, func(q queryer) error {
		items, err := listExperiments(ctx, q, req)
		result = items
		return err
	})
	return result, err
}

func listExperiments(ctx context.Context, q queryer, req *kmapv1.ListExperimentsRequest) ([]*kmapv1.ExperimentSummary, error) {
	const sql = `
SELECT f.id::text, s.canonical_name, p.canonical_name, coalesce(f.conditions, '{}'::jsonb),
       f.value_raw, d.title, d.doc_type::text, f.extraction_confidence
FROM kg.numeric_facts f
JOIN kg.entities s ON s.id = f.subject_id
JOIN kg.entities p ON p.id = f.parameter_id
JOIN core.documents d ON d.id = f.document_id
WHERE ($1 = '' OR s.slug = $1 OR s.canonical_name ILIKE '%' || $1 || '%')
  AND ($2 = '' OR p.slug = $2 OR p.canonical_name ILIKE '%' || $2 || '%')
  AND ($3 = 0 OR d.year >= $3)
ORDER BY d.year DESC NULLS LAST, f.extraction_confidence DESC
LIMIT 50`
	rows, err := q.Query(ctx, sql, req.GetProcess(), req.GetParameter(), req.GetYearFrom())
	if err != nil {
		return nil, fmt.Errorf("query experiments: %w", err)
	}
	defer rows.Close()

	var result []*kmapv1.ExperimentSummary
	for rows.Next() {
		item := &kmapv1.ExperimentSummary{}
		var conditions []byte
		if err := rows.Scan(&item.Id, &item.Process, &item.Material, &conditions, &item.Result, &item.Source, &item.DocType, &item.Confidence); err != nil {
			return nil, fmt.Errorf("scan experiment: %w", err)
		}
		item.Code = item.Id
		item.Conditions = bytesToStruct(conditions)
		result = append(result, item)
	}
	return result, rows.Err()
}

func entityAliases(ctx context.Context, q queryer, entityID string) []string {
	const sql = `SELECT alias FROM kg.entity_aliases WHERE entity_id::text = $1 ORDER BY alias LIMIT 20`
	rows, err := q.Query(ctx, sql, entityID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []string
	for rows.Next() {
		var alias string
		if err := rows.Scan(&alias); err != nil {
			return result
		}
		result = append(result, alias)
	}
	return result
}

func bytesToStruct(raw []byte) *structpb.Struct {
	values := map[string]any{}
	_ = json.Unmarshal(raw, &values)
	result, err := structpb.NewStruct(values)
	if err != nil {
		return &structpb.Struct{}
	}
	return result
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
