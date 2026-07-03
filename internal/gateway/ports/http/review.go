package http

import (
	"context"
	stdhttp "net/http"

	"github.com/jackc/pgx/v5"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/audit"
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

type reviewContradiction struct {
	ID        string  `json:"id"`
	ClusterID string  `json:"clusterId"`
	Status    string  `json:"status"`
	Dtype     string  `json:"dtype"`
	Severity  float64 `json:"severity"`
	Rationale string  `json:"rationale"`
}

type entityStatusBody struct {
	Status  string `json:"status"`
	Comment string `json:"comment"`
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

const reviewFactsSQL = `
SELECT f.id::text, f.value_raw, coalesce(f.quote, ''), f.validation_status::text, p.slug
FROM kg.numeric_facts f
JOIN kg.entities p ON p.id = f.parameter_id
WHERE f.validation_status = 'contradicted' AND f.superseded_by IS NULL
ORDER BY f.created_at DESC
LIMIT 50`

const reviewContradictionsSQL = `
SELECT id::text, coalesce(cluster_id::text, ''), status, coalesce(dtype, ''),
       coalesce(severity, 0), coalesce(judge_rationale, '')
FROM epi.contradictions
WHERE status IN ('suspected', 'judge_confirmed') AND decided_at IS NULL
ORDER BY severity DESC, created_at DESC
LIMIT 50`

const updateEntityStatusSQL = `
UPDATE kg.entities SET status = $2, updated_at = now()
WHERE id = $1 AND status = 'pending_review'`

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
		items, err := server.reviewOrphans(r.Context(), rls, reviewOrphansSQL)
		if err != nil {
			writeProblem(w, r, stdhttp.StatusInternalServerError, "internal", "Internal server error", err.Error())
			return
		}
		writeDataJSON(w, stdhttp.StatusOK, itemsResponse[reviewOrphan]{Items: items})
	case "facts":
		items, err := server.reviewOrphans(r.Context(), rls, reviewFactsSQL)
		if err != nil {
			writeProblem(w, r, stdhttp.StatusInternalServerError, "internal", "Internal server error", err.Error())
			return
		}
		writeDataJSON(w, stdhttp.StatusOK, itemsResponse[reviewOrphan]{Items: items})
	case "contradictions":
		items, err := server.reviewContradictions(r.Context(), rls)
		if err != nil {
			writeProblem(w, r, stdhttp.StatusInternalServerError, "internal", "Internal server error", err.Error())
			return
		}
		writeDataJSON(w, stdhttp.StatusOK, itemsResponse[reviewContradiction]{Items: items})
	default:
		writeProblem(w, r, stdhttp.StatusBadRequest, "invalid_request", "Invalid request", "kind must be entities, facts, contradictions or orphans")
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

func (server *Server) reviewOrphans(ctx context.Context, rls pg.Principal, query string) ([]reviewOrphan, error) {
	items := make([]reviewOrphan, 0)
	err := pg.WithRLS(ctx, server.pool.Pool, rls, func(ctx context.Context, tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query)
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

func (server *Server) reviewContradictions(ctx context.Context, rls pg.Principal) ([]reviewContradiction, error) {
	items := make([]reviewContradiction, 0)
	err := pg.WithRLS(ctx, server.pool.Pool, rls, func(ctx context.Context, tx pgx.Tx) error {
		rows, err := tx.Query(ctx, reviewContradictionsSQL)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item reviewContradiction
			if err := rows.Scan(&item.ID, &item.ClusterID, &item.Status, &item.Dtype, &item.Severity, &item.Rationale); err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}

func (server *Server) updateEntityStatusHandler(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var body entityStatusBody
	if !readJSONBody(w, r, &body) {
		return
	}
	target := map[string]string{"accept": "active", "reject": "deprecated"}[body.Status]
	if target == "" {
		writeProblem(w, r, stdhttp.StatusBadRequest, "invalid_request", "Invalid request", "status must be accept or reject")
		return
	}
	entityID := pathParam(r, "id")
	principal := principalFromContext(r)
	rls := pg.Principal{UserID: principal.GetUserId(), DocAccess: principal.GetDocAccess()}
	var affected int64
	err := pg.WithRLS(r.Context(), server.pool.Pool, rls, func(ctx context.Context, tx pgx.Tx) error {
		tag, execErr := tx.Exec(ctx, updateEntityStatusSQL, entityID, target)
		if execErr != nil {
			return execErr
		}
		affected = tag.RowsAffected()
		return nil
	})
	if err != nil {
		writeProblem(w, r, stdhttp.StatusInternalServerError, "internal", "Internal server error", err.Error())
		return
	}
	if affected == 0 {
		writeProblem(w, r, stdhttp.StatusNotFound, "not_found", "Not found", "pending entity not found")
		return
	}
	if server.audit != nil {
		_ = server.audit.Write(r.Context(), audit.Record{
			ActorID:    principal.GetUserId(),
			Action:     "entity.review",
			ObjectType: "entity",
			ObjectID:   entityID,
			RequestID:  r.Header.Get("X-Request-Id"),
			IP:         clientIP(r),
			Details:    map[string]any{"status": target, "comment": body.Comment},
		})
	}
	writeDataJSON(w, stdhttp.StatusOK, map[string]any{"id": entityID, "status": target})
}
