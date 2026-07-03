package http

import (
	"context"
	stdhttp "net/http"

	"github.com/jackc/pgx/v5"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/pg"
)

type reviewEntity struct {
	ID         string  `json:"id"`
	Slug       string  `json:"slug"`
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	Candidate  string  `json:"candidate"`
	Similarity float64 `json:"similarity"`
}

type reviewOrphan struct {
	ID         string   `json:"id"`
	Value      string   `json:"value"`
	Quote      string   `json:"quote"`
	Status     string   `json:"status"`
	Candidates []string `json:"candidates"`
}

const reviewEntitiesSQL = `
SELECT e.id::text, e.slug, e.canonical_name, e.etype::text,
       coalesce(c.candidate, ''), coalesce(c.similarity, 0)
FROM kg.entities e
LEFT JOIN LATERAL (
  SELECT a.canonical_name || ' (' || a.slug || ')' AS candidate,
         similarity(a.canonical_name, e.canonical_name) AS similarity
  FROM kg.entities a
  WHERE a.status = 'active' AND a.etype = e.etype AND a.id <> e.id
  ORDER BY similarity(a.canonical_name, e.canonical_name) DESC
  LIMIT 1
) c ON true
WHERE e.status = 'pending_review'
ORDER BY e.created_at DESC
LIMIT 50`

const reviewOrphansSQL = `
SELECT f.id::text, f.value_raw, coalesce(f.quote, ''), f.validation_status::text, p.slug
FROM kg.numeric_facts f
JOIN kg.entities p ON p.id = f.parameter_id
WHERE f.validation_status IN ('needs_unit_review', 'weak_evidence') AND f.superseded_by IS NULL
ORDER BY f.created_at DESC
LIMIT 50`

func (server *Server) reviewQueueHandler(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	principal := principalFromContext(r)
	rls := pg.Principal{UserID: principal.GetUserId(), DocAccess: principal.GetDocAccess()}

	switch r.URL.Query().Get("kind") {
	case "entities":
		items, err := server.reviewEntities(r.Context(), rls)
		if err != nil {
			writeProblem(w, r, stdhttp.StatusInternalServerError, "internal", "Internal server error", err.Error())
			return
		}
		writeDataJSON(w, stdhttp.StatusOK, itemsResponse[reviewEntity]{Items: items})
	case "orphans":
		items, err := server.reviewOrphans(r.Context(), rls)
		if err != nil {
			writeProblem(w, r, stdhttp.StatusInternalServerError, "internal", "Internal server error", err.Error())
			return
		}
		writeDataJSON(w, stdhttp.StatusOK, itemsResponse[reviewOrphan]{Items: items})
	default:
		writeProblem(w, r, stdhttp.StatusBadRequest, "invalid_request", "Invalid request", "kind must be entities or orphans")
	}
}

func (server *Server) reviewEntities(ctx context.Context, rls pg.Principal) ([]reviewEntity, error) {
	items := make([]reviewEntity, 0)
	err := pg.WithRLS(ctx, server.pool.Pool, rls, func(ctx context.Context, tx pgx.Tx) error {
		rows, err := tx.Query(ctx, reviewEntitiesSQL)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item reviewEntity
			if err := rows.Scan(&item.ID, &item.Slug, &item.Name, &item.Type, &item.Candidate, &item.Similarity); err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}

func (server *Server) reviewOrphans(ctx context.Context, rls pg.Principal) ([]reviewOrphan, error) {
	items := make([]reviewOrphan, 0)
	err := pg.WithRLS(ctx, server.pool.Pool, rls, func(ctx context.Context, tx pgx.Tx) error {
		rows, err := tx.Query(ctx, reviewOrphansSQL)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item reviewOrphan
			var candidate string
			if err := rows.Scan(&item.ID, &item.Value, &item.Quote, &item.Status, &candidate); err != nil {
				return err
			}
			item.Candidates = []string{candidate}
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}
