package pg

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maksim-mshp/nornickel-hackathon/internal/ingest/domain"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/events"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/outbox"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const findIDBySHA256SQL = `select id from core.documents where sha256 = $1`

func (repository *Repository) FindIDBySHA256(ctx context.Context, sha256 []byte) (uuid.UUID, bool, error) {
	var id uuid.UUID
	err := repository.pool.QueryRow(ctx, findIDBySHA256SQL, sha256).Scan(&id)
	if err == pgx.ErrNoRows {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("find document by sha256: %w", err)
	}
	return id, true, nil
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
	)
	err := repository.pool.QueryRow(ctx, getDocumentSQL, documentID).Scan(&status, &version)
	if err == pgx.ErrNoRows {
		return domain.Document{}, nil, domain.ErrDocumentNotFound
	}
	if err != nil {
		return domain.Document{}, nil, fmt.Errorf("get document status: %w", err)
	}

	rows, err := repository.pool.Query(ctx, getStagesSQL, documentID, version)
	if err != nil {
		return domain.Document{}, nil, fmt.Errorf("query ingest stages: %w", err)
	}
	defer rows.Close()

	var stages []domain.Stage
	for rows.Next() {
		var stage domain.Stage
		if err := rows.Scan(&stage.Stage, &stage.Status, &stage.Attempt, &stage.Error); err != nil {
			return domain.Document{}, nil, fmt.Errorf("scan ingest stage: %w", err)
		}
		stages = append(stages, stage)
	}
	if err := rows.Err(); err != nil {
		return domain.Document{}, nil, fmt.Errorf("iterate ingest stages: %w", err)
	}

	doc := domain.Document{ID: documentID, Version: version, Status: status}
	return doc, stages, nil
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
