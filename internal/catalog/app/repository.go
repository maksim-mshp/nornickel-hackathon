package app

import (
	"context"

	"github.com/google/uuid"
	"github.com/maksim-mshp/nornickel-hackathon/internal/catalog/domain"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/events"
)

type ChunkInsert struct {
	UUID  uuid.UUID
	Chunk domain.Chunk
}

type CommitCommand struct {
	DocumentID  uuid.UUID
	Version     int
	NewEntities []domain.Entity
	Chunks      []ChunkInsert
	Facts       []domain.NumericFact
}

type Repository interface {
	ResolveByNames(ctx context.Context, names []string) (map[string]uuid.UUID, error)
	ParameterDefs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]domain.ParameterDef, error)
	Commit(ctx context.Context, cmd CommitCommand, committed events.Envelope, clusterDirty events.Envelope) error
	UpdateFactStatus(ctx context.Context, factID string, factKind string, status string, actor string, comment string) error
	MergeEntities(ctx context.Context, entityID string, intoID string, actor string, comment string) error
}
