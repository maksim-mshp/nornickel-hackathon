package pg

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"github.com/maksim-mshp/nornickel-hackathon/internal/answer/app"
	"google.golang.org/protobuf/types/known/structpb"
)

type Retriever struct {
	pool *pgxpool.Pool
}

func NewRetriever(pool *pgxpool.Pool) *Retriever {
	return &Retriever{pool: pool}
}

const resolveEntitySQL = `
SELECT slug, canonical_name, etype::text
FROM kg.entities
WHERE status IN ('active', 'pending_review')
  AND char_length(canonical_name) >= 3
  AND (canonical_name ILIKE '%' || $1 || '%'
       OR $1 ILIKE '%' || canonical_name || '%'
       OR similarity(canonical_name, $1) > 0.4)
ORDER BY (lower(canonical_name) = lower($1)) DESC, similarity(canonical_name, $1) DESC
LIMIT 1`

func (retriever *Retriever) ResolveEntities(ctx context.Context, terms []string) ([]app.ResolvedEntity, error) {
	resolved := make([]app.ResolvedEntity, 0, len(terms))
	seen := map[string]bool{}
	for _, term := range terms {
		term = strings.TrimSpace(term)
		if term == "" {
			continue
		}
		var slug, name, etype string
		err := retriever.pool.QueryRow(ctx, resolveEntitySQL, term).Scan(&slug, &name, &etype)
		if err != nil {
			continue
		}
		if slug == "" || seen[slug] {
			continue
		}
		seen[slug] = true
		resolved = append(resolved, app.ResolvedEntity{Slug: slug, Name: name, Etype: etype})
	}
	return resolved, nil
}

const factsByTextSQL = `
WITH q AS (
  SELECT websearch_to_tsquery('russian', $1) || websearch_to_tsquery('english', $1)
       || websearch_to_tsquery('russian', $2) || websearch_to_tsquery('english', $2) AS query
),
cr AS (
  SELECT c.id AS chunk_id, c.document_id, ts_rank_cd(c.tsv_ru || c.tsv_en, q.query) AS crank
  FROM core.chunks c, q
  WHERE c.tsv_ru @@ q.query OR c.tsv_en @@ q.query
),
dr AS (
  SELECT document_id, max(crank) AS chunk_rank
  FROM cr
  GROUP BY document_id
  ORDER BY chunk_rank DESC
  LIMIT 40
),
drt AS (
  SELECT dr.document_id,
         dr.chunk_rank + 3 * ts_rank(to_tsvector('russian', d.title) || to_tsvector('english', d.title), q.query) AS drank
  FROM dr
  JOIN core.documents d ON d.id = dr.document_id, q
  ORDER BY drank DESC
  LIMIT 12
),
picked AS (
  SELECT f.id, f.operator, f.vmin, f.vmax, f.unit_orig, f.vmin_si, f.vmax_si, f.unit_code,
         f.parameter_id, f.relation, f.geography, f.quote, f.page, f.document_id,
         f.extraction_method, f.extractor_version, f.extraction_confidence, f.validation_status,
         drt.drank, coalesce(cr.crank, 0) AS crank,
         row_number() OVER (
           PARTITION BY f.document_id
           ORDER BY coalesce(cr.crank, 0) DESC, f.extraction_confidence DESC
         ) AS rn
  FROM drt
  JOIN kg.numeric_facts f ON f.document_id = drt.document_id AND f.unit_code IS NOT NULL
  LEFT JOIN cr ON cr.chunk_id = f.chunk_id
)
SELECT p.id::text, p.operator::text, p.vmin::float8, p.vmax::float8, coalesce(p.unit_orig, ''),
       p.vmin_si::float8, p.vmax_si::float8, coalesce(u.si_unit, ''),
       coalesce(pe.canonical_name, p.relation), coalesce(pe.slug, ''),
       p.geography::text, coalesce(p.quote, ''), coalesce(p.page, 0),
       coalesce(d.title, ''), coalesce(d.doc_type::text, ''), coalesce(d.year, 0),
       p.extraction_method::text, coalesce(p.extractor_version, ''),
       p.extraction_confidence::float8, p.validation_status::text, p.document_id::text
FROM picked p
JOIN core.documents d ON d.id = p.document_id
LEFT JOIN kg.entities pe ON pe.id = p.parameter_id
LEFT JOIN kg.units u ON u.code = p.unit_code
WHERE p.rn <= 6
ORDER BY p.crank DESC, p.drank DESC, p.extraction_confidence DESC
LIMIT $3`

type factRow struct {
	id, operator, unitOrig, siUnit    string
	paramName, paramSlug, geography   string
	quote, docTitle, docType, docID   string
	method, version, validationStatus string
	vmin, vmax, vminSi, vmaxSi        sql.NullFloat64
	page, year                        int
	confidence                        float64
}

func (retriever *Retriever) FactsByText(ctx context.Context, termsQuery string, question string, limit int) ([]*kmapv1.Fact, error) {
	termsQuery = strings.TrimSpace(termsQuery)
	question = strings.TrimSpace(question)
	if (termsQuery == "" && question == "") || limit <= 0 {
		return nil, nil
	}
	rows, err := retriever.pool.Query(ctx, factsByTextSQL, termsQuery, question, limit)
	if err != nil {
		return nil, fmt.Errorf("facts by text: %w", err)
	}
	defer rows.Close()

	facts := make([]*kmapv1.Fact, 0, limit)
	index := 0
	for rows.Next() {
		var row factRow
		if err := rows.Scan(
			&row.id, &row.operator, &row.vmin, &row.vmax, &row.unitOrig,
			&row.vminSi, &row.vmaxSi, &row.siUnit,
			&row.paramName, &row.paramSlug,
			&row.geography, &row.quote, &row.page,
			&row.docTitle, &row.docType, &row.year,
			&row.method, &row.version,
			&row.confidence, &row.validationStatus, &row.docID,
		); err != nil {
			return nil, fmt.Errorf("scan fact: %w", err)
		}
		index++
		payload, err := factPayload(row, fmt.Sprintf("F%d", index))
		if err != nil {
			return nil, err
		}
		facts = append(facts, &kmapv1.Fact{Id: row.id, Kind: "numeric", Payload: payload})
	}
	return facts, rows.Err()
}

func factPayload(row factRow, ref string) (*structpb.Struct, error) {
	value := map[string]any{"operator": row.operator, "unit": row.unitOrig}
	si := map[string]any{"operator": row.operator, "unit": row.siUnit}
	if row.vmin.Valid {
		value["vmin"] = row.vmin.Float64
	}
	if row.vmax.Valid {
		value["vmax"] = row.vmax.Float64
	}
	if row.vminSi.Valid {
		si["vmin"] = row.vminSi.Float64
	}
	if row.vmaxSi.Valid {
		si["vmax"] = row.vmaxSi.Float64
	}
	payload := map[string]any{
		"id":        row.id,
		"ref":       ref,
		"subject":   map[string]any{"slug": "", "name": row.docTitle},
		"parameter": map[string]any{"slug": row.paramSlug, "name": row.paramName},
		"value":     value,
		"si":        si,
		"geography": row.geography,
		"provenance": map[string]any{
			"documentId": row.docID,
			"title":      row.docTitle,
			"docType":    row.docType,
			"page":       float64(row.page),
			"quote":      row.quote,
			"year":       float64(row.year),
		},
		"extractionMethod": row.method,
		"extractorVersion": row.version,
		"confidence":       row.confidence,
		"validationStatus": row.validationStatus,
	}
	return structpb.NewStruct(payload)
}
