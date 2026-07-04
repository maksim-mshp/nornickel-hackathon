package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	stdhttp "net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/audit"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/blob"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/pg"
)

var sourceMIME = map[string]string{
	".pdf":  "application/pdf",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".doc":  "application/msword",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".xls":  "application/vnd.ms-excel",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".ppt":  "application/vnd.ms-powerpoint",
	".csv":  "text/csv; charset=utf-8",
	".txt":  "text/plain; charset=utf-8",
	".rtf":  "application/rtf",
	".htm":  "text/html; charset=utf-8",
	".html": "text/html; charset=utf-8",
}

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

	filename := sourceFilename(title, key)
	w.Header().Set("Content-Type", contentTypeFor(filename, head))
	w.Header().Set("Content-Disposition", contentDisposition(filename))
	w.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
	w.WriteHeader(stdhttp.StatusOK)
	if _, err := w.Write(head); err != nil {
		return
	}
	_, _ = io.Copy(w, object)
}

func sourceFilename(title string, blobKey string) string {
	name := strings.TrimSpace(title)
	if name == "" {
		name = "document"
	}
	if filepath.Ext(name) == "" {
		if ext := filepath.Ext(blobKey); ext != "" {
			name += ext
		}
	}
	return name
}

func contentTypeFor(filename string, head []byte) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if mediaType, ok := sourceMIME[ext]; ok {
		return mediaType
	}
	if mediaType := mime.TypeByExtension(ext); mediaType != "" {
		return mediaType
	}
	return stdhttp.DetectContentType(head)
}

func contentDisposition(filename string) string {
	return fmt.Sprintf("attachment; filename=%q; filename*=UTF-8''%s", asciiFilename(filename), rfc5987Escape(filename))
}

func asciiFilename(name string) string {
	var builder strings.Builder
	for _, symbol := range name {
		if symbol < 0x20 || symbol > 0x7e || symbol == '"' || symbol == '\\' {
			builder.WriteByte('_')
			continue
		}
		builder.WriteRune(symbol)
	}
	fallback := strings.TrimSpace(builder.String())
	if fallback == "" {
		return "document"
	}
	return fallback
}

func rfc5987Escape(name string) string {
	const upperhex = "0123456789ABCDEF"
	var builder strings.Builder
	for index := 0; index < len(name); index++ {
		char := name[index]
		if isAttrChar(char) {
			builder.WriteByte(char)
			continue
		}
		builder.WriteByte('%')
		builder.WriteByte(upperhex[char>>4])
		builder.WriteByte(upperhex[char&0x0f])
	}
	return builder.String()
}

func isAttrChar(char byte) bool {
	switch {
	case char >= 'A' && char <= 'Z', char >= 'a' && char <= 'z', char >= '0' && char <= '9':
		return true
	}
	switch char {
	case '!', '#', '$', '&', '+', '-', '.', '^', '_', '`', '|', '~':
		return true
	}
	return false
}
