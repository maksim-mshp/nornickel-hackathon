package pg

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maksim-mshp/nornickel-hackathon/internal/ingest/app"
	"github.com/maksim-mshp/nornickel-hackathon/internal/ingest/domain"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/auth"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/events"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/outbox"
	platformpg "github.com/maksim-mshp/nornickel-hackathon/internal/platform/pg"
)

func rlsPrincipal(ctx context.Context) platformpg.Principal {
	principal, _ := auth.FromContext(ctx)
	return platformpg.Principal{UserID: principal.UserID, DocAccess: principal.DocAccess}
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const findIDBySHA256SQL = `select id from core.documents where sha256 = $1`

func (repository *Repository) FindIDBySHA256(ctx context.Context, sha256 []byte) (uuid.UUID, bool, error) {
	var id uuid.UUID
	found := false
	err := platformpg.WithRLS(ctx, repository.pool, platformpg.Principal{UserID: "system", DocAccess: auth.AccessRestricted}, func(ctx context.Context, tx pgx.Tx) error {
		scanErr := tx.QueryRow(ctx, findIDBySHA256SQL, sha256).Scan(&id)
		if scanErr == pgx.ErrNoRows {
			return nil
		}
		if scanErr != nil {
			return fmt.Errorf("find document by sha256: %w", scanErr)
		}
		found = true
		return nil
	})
	if err != nil {
		return uuid.Nil, false, err
	}
	return id, found, nil
}

const insertDocumentSQL = `insert into core.documents
(id, title, doc_type, lang, year, geography, access_level, source_uri, sha256, status, current_version, uploaded_by, meta)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

const insertVersionSQL = `insert into core.document_versions
(document_id, version, blob_uri) values ($1, $2, $3)`

const upsertStageSQL = `insert into ops.ingest_jobs (document_id, version, stage, status)
values ($1, $2, $3, $4)
on conflict (document_id, version, stage) do update set status = excluded.status, finished_at = now()`

func (repository *Repository) Register(ctx context.Context, doc domain.Document, envelope events.Envelope) (domain.Document, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return domain.Document{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := platformpg.SetRLS(ctx, tx, rlsPrincipal(ctx)); err != nil {
		return domain.Document{}, err
	}

	if _, err := tx.Exec(ctx, insertDocumentSQL,
		doc.ID, doc.Title, doc.DocType, nullableString(doc.Lang), nullableInt(doc.Year),
		doc.Geography, doc.AccessLevel, doc.SourceURI, doc.SHA256, doc.Status, doc.Version,
		uploadedBy(doc.UploadedBy), nullableJSON(doc.Meta),
	); err != nil {
		return domain.Document{}, fmt.Errorf("insert document: %w", err)
	}

	if _, err := tx.Exec(ctx, insertVersionSQL, doc.ID, doc.Version, doc.BlobURI); err != nil {
		return domain.Document{}, fmt.Errorf("insert document version: %w", err)
	}

	for _, stage := range domain.DefaultStages() {
		if _, err := tx.Exec(ctx, upsertStageSQL, doc.ID, doc.Version, stage.Stage, stage.Status); err != nil {
			return domain.Document{}, fmt.Errorf("upsert stage %q: %w", stage.Stage, err)
		}
	}

	if err := outbox.Append(ctx, tx, outbox.Record{
		Envelope:      envelope,
		AggregateType: "document",
		AggregateID:   &doc.ID,
	}); err != nil {
		return domain.Document{}, fmt.Errorf("append outbox: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Document{}, fmt.Errorf("commit tx: %w", err)
	}
	return doc, nil
}

const getDocumentSQL = `select status, current_version from core.documents where id = $1`
const getStagesSQL = `select stage, status, attempt, coalesce(error, '')
from ops.ingest_jobs
where document_id = $1 and version = $2
order by array_position(array['register','parse','extract','commit','epistemic'], stage)`

func (repository *Repository) GetStatus(ctx context.Context, documentID uuid.UUID) (domain.Document, []domain.Stage, error) {
	var (
		status  string
		version int
		stages  []domain.Stage
	)
	err := platformpg.WithRLS(ctx, repository.pool, rlsPrincipal(ctx), func(ctx context.Context, tx pgx.Tx) error {
		scanErr := tx.QueryRow(ctx, getDocumentSQL, documentID).Scan(&status, &version)
		if scanErr == pgx.ErrNoRows {
			return domain.ErrDocumentNotFound
		}
		if scanErr != nil {
			return fmt.Errorf("get document status: %w", scanErr)
		}

		rows, scanErr := tx.Query(ctx, getStagesSQL, documentID, version)
		if scanErr != nil {
			return fmt.Errorf("query ingest stages: %w", scanErr)
		}
		defer rows.Close()

		for rows.Next() {
			var stage domain.Stage
			if scanErr := rows.Scan(&stage.Stage, &stage.Status, &stage.Attempt, &stage.Error); scanErr != nil {
				return fmt.Errorf("scan ingest stage: %w", scanErr)
			}
			stages = append(stages, stage)
		}
		return rows.Err()
	})
	if err != nil {
		return domain.Document{}, nil, err
	}

	doc := domain.Document{ID: documentID, Version: version, Status: status}
	return doc, stages, nil
}

func (repository *Repository) ListDocuments(ctx context.Context, limit uint32) ([]app.DocumentSummary, error) {
	const query = `
SELECT d.id, d.title, d.doc_type::text, coalesce(d.lang, ''), d.geography::text,
       d.access_level::text, d.status::text, count(f.id)::int, coalesce(d.year, 0)
FROM core.documents d
LEFT JOIN kg.numeric_facts f ON f.document_id = d.id
GROUP BY d.id, d.title, d.doc_type, d.lang, d.geography, d.access_level, d.status, d.year, d.created_at
ORDER BY d.created_at DESC
LIMIT $1`
	var result []app.DocumentSummary
	err := platformpg.WithRLS(ctx, repository.pool, rlsPrincipal(ctx), func(ctx context.Context, tx pgx.Tx) error {
		rows, queryErr := tx.Query(ctx, query, int(limit))
		if queryErr != nil {
			return fmt.Errorf("query documents: %w", queryErr)
		}
		defer rows.Close()
		for rows.Next() {
			var item app.DocumentSummary
			if scanErr := rows.Scan(&item.ID, &item.Title, &item.DocType, &item.Lang, &item.Geography, &item.AccessLevel, &item.Status, &item.Facts, &item.Year); scanErr != nil {
				return fmt.Errorf("scan document: %w", scanErr)
			}
			result = append(result, item)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableInt(value int) any {
	if value == 0 {
		return nil
	}
	return value
}

func nullableJSON(value map[string]any) any {
	if len(value) == 0 {
		return []byte("{}")
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}
	return raw
}

func uploadedBy(value string) any {
	if value == "" {
		return nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return nil
	}
	return id
}
