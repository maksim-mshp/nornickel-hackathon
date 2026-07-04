package app

import (
	"context"

	"github.com/google/uuid"
	"github.com/maksim-mshp/nornickel-hackathon/internal/ingest/domain"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/events"
)

type Repository interface {
	FindIDBySHA256(ctx context.Context, sha256 []byte) (uuid.UUID, bool, error)
	Register(ctx context.Context, doc domain.Document, envelope events.Envelope) (domain.Document, bool, error)
	GetStatus(ctx context.Context, documentID uuid.UUID) (domain.Document, []domain.Stage, error)
	ListDocuments(ctx context.Context, offset uint32, limit uint32) ([]DocumentSummary, uint32, error)
}

type DocumentSummary struct {
	ID            uuid.UUID
	Title         string
	DocType       string
	Lang          string
	Geography     string
	AccessLevel   string
	Status        string
	Facts         uint32
	Year          int32
	Version       int32
	NcSuspectRate float64
}
