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
ranked AS (
  SELECT c.id, ts_rank_cd(c.tsv_ru || c.tsv_en, q.query) AS rank
  FROM core.chunks c, q
  WHERE (c.tsv_ru || c.tsv_en) @@ q.query
  ORDER BY rank DESC
  LIMIT 200
)
SELECT f.id::text, f.operator::text, f.vmin::float8, f.vmax::float8, coalesce(f.unit_orig, ''),
       f.vmin_si::float8, f.vmax_si::float8, coalesce(u.si_unit, ''),
       coalesce(p.canonical_name, f.relation), coalesce(p.slug, ''),
       f.geography::text, coalesce(f.quote, ''), coalesce(f.page, 0),
       coalesce(d.title, ''), coalesce(d.doc_type::text, ''), coalesce(d.year, 0),
       f.extraction_method::text, coalesce(f.extractor_version, ''),
       f.extraction_confidence::float8, f.validation_status::text
FROM ranked r
JOIN kg.numeric_facts f ON f.chunk_id = r.id
JOIN core.documents d ON d.id = f.document_id
LEFT JOIN kg.entities p ON p.id = f.parameter_id
LEFT JOIN kg.units u ON u.code = f.unit_code
WHERE f.unit_code IS NOT NULL
ORDER BY r.rank DESC, f.extraction_confidence DESC
LIMIT $3`

type factRow struct {
	id, operator, unitOrig, siUnit    string
	paramName, paramSlug, geography   string
	quote, docTitle, docType          string
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
			&row.confidence, &row.validationStatus,
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
			"documentId": "",
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
