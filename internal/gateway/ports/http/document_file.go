package http

import (
	"context"
	"errors"
	"io"
	"mime"
	stdhttp "net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/audit"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/blob"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/pg"
)

const documentFileSQL = `select d.title, coalesce(dv.blob_uri, '')
from core.documents d
join core.document_versions dv on dv.document_id = d.id and dv.version = d.current_version
where d.id = $1`

func (server *Server) documentFileHandler(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	documentID := chi.URLParam(r, "document_id")
	if documentID == "" {
		writeProblem(w, r, stdhttp.StatusBadRequest, "invalid_request", "Invalid request", "document_id is required")
		return
	}

	principal := principalFromContext(r)
	var title, blobURI string
	err := pg.WithRLS(r.Context(), server.pool.Pool, pg.Principal{
		UserID:    principal.GetUserId(),
		DocAccess: principal.GetDocAccess(),
	}, func(ctx context.Context, tx pgx.Tx) error {
		return tx.QueryRow(ctx, documentFileSQL, documentID).Scan(&title, &blobURI)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeProblem(w, r, stdhttp.StatusNotFound, "not_found", "Document not found", "")
		return
	}
	if err != nil {
		writeProblem(w, r, stdhttp.StatusInternalServerError, "internal", "Internal server error", err.Error())
		return
	}
	if blobURI == "" {
		writeProblem(w, r, stdhttp.StatusNotFound, "not_found", "Document has no stored source file", "")
		return
	}

	bucket, key, err := blob.ParseURI(blobURI)
	if err != nil {
		writeProblem(w, r, stdhttp.StatusInternalServerError, "internal", "Invalid blob uri", err.Error())
		return
	}

	object, err := server.blob.Get(r.Context(), bucket, key)
	if err != nil {
		writeProblem(w, r, stdhttp.StatusBadGateway, "blob_unavailable", "Source file unavailable", err.Error())
		return
	}
	defer func() {
		_ = object.Close()
	}()

	if server.audit != nil {
		_ = server.audit.Write(r.Context(), audit.Record{
			ActorID:    principal.GetUserId(),
			Action:     "document.view_source",
			ObjectType: "document",
			ObjectID:   documentID,
			RequestID:  r.Header.Get("X-Request-Id"),
			IP:         clientIP(r),
			Details:    map[string]any{"roles": principal.GetRoles()},
		})
	}

	head := make([]byte, 512)
	n, readErr := io.ReadFull(object, head)
	if readErr != nil && !errors.Is(readErr, io.EOF) && !errors.Is(readErr, io.ErrUnexpectedEOF) {
		writeProblem(w, r, stdhttp.StatusNotFound, "not_found", "Source file missing in storage", readErr.Error())
		return
	}
	head = head[:n]

	filename := title
	if filename == "" {
		filename = documentID
	}
	w.Header().Set("Content-Type", stdhttp.DetectContentType(head))
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": filename}))
	w.WriteHeader(stdhttp.StatusOK)
	if _, err := w.Write(head); err != nil {
		return
	}
	_, _ = io.Copy(w, object)
}
