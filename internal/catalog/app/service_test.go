package app

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/maksim-mshp/nornickel-hackathon/internal/catalog/domain"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/blob"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/events"
)

type fakeRepository struct {
	resolved     map[string]uuid.UUID
	defs         map[uuid.UUID]domain.ParameterDef
	committed    CommitCommand
	factsEvent   events.Envelope
	clusterEvent events.Envelope
}

func (repo *fakeRepository) ResolveByNames(_ context.Context, _ []string) (map[string]uuid.UUID, error) {
	if repo.resolved == nil {
		return map[string]uuid.UUID{}, nil
	}
	return repo.resolved, nil
}

func (repo *fakeRepository) ParameterDefs(context.Context, []uuid.UUID) (map[uuid.UUID]domain.ParameterDef, error) {
	if repo.defs == nil {
		return map[uuid.UUID]domain.ParameterDef{}, nil
	}
	return repo.defs, nil
}

func (repo *fakeRepository) Commit(_ context.Context, cmd CommitCommand, committed events.Envelope, clusterDirty events.Envelope) error {
	repo.committed = cmd
	repo.factsEvent = committed
	repo.clusterEvent = clusterDirty
	return nil
}

func (repo *fakeRepository) UpdateFactStatus(context.Context, string, string, string, string, string) error {
	return nil
}

func (repo *fakeRepository) MergeEntities(context.Context, string, string, string, string) error {
	return nil
}

func (repo *fakeRepository) MarkDocumentFailed(context.Context, uuid.UUID, string) error {
	return nil
}

func TestCommitExtractionAcceptsNumericCandidatesBundle(t *testing.T) {
	t.Parallel()

	documentID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{}
	service, bundleURI := serviceWithBundle(t, repo, map[string]any{
		"schema":            "extraction-bundle/v2",
		"document_id":       documentID.String(),
		"version":           3,
		"extractor_version": "extract-test",
		"chunks": []map[string]any{
			{"id": "c1", "ordinal": 1, "text": "quote", "kind": "text"},
		},
		"entities": []map[string]any{
			{"type": "process", "name": "Nickel electrowinning"},
		},
		"numeric_candidates": []map[string]any{
			{"subject": "Nickel electrowinning", "parameter": "temperature", "operator": "range", "vmin": 60, "vmax": 80, "unit_orig": "C", "unit_code": "degC", "quote": "60-80 C", "chunk_id": "c1", "confidence": 0.9},
		},
	})

	result, err := service.CommitExtraction(t.Context(), bundleURI)
	if err != nil {
		t.Fatalf("commit extraction: %v", err)
	}

	if result.DocumentID != documentID {
		t.Fatalf("expected document id %s, got %s", documentID, result.DocumentID)
	}
	if repo.committed.Version != 3 {
		t.Fatalf("expected bundle version 3, got %d", repo.committed.Version)
	}
	if len(repo.committed.Facts) != 1 {
		t.Fatalf("expected one fact, got %d", len(repo.committed.Facts))
	}
	if repo.committed.Facts[0].ChunkID == nil {
		t.Fatal("expected candidate chunk reference to be mapped")
	}
	if repo.factsEvent.Type != events.FactsCommitted {
		t.Fatalf("expected facts event, got %q", repo.factsEvent.Type)
	}
	if repo.clusterEvent.Type != events.EpistemicClusterDirty {
		t.Fatalf("expected cluster dirty event, got %q", repo.clusterEvent.Type)
	}
}

func TestCommitExtractionAcceptsLegacyNumericFactsBundle(t *testing.T) {
	t.Parallel()

	documentID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{}
	service, bundleURI := serviceWithBundle(t, repo, map[string]any{
		"document_id":       documentID.String(),
		"extractor_version": "legacy-extract",
		"chunks": []map[string]any{
			{"ordinal": 1, "text": "quote", "kind": "text"},
		},
		"entities": []map[string]any{
			{"slug": "process:nickel-electrowinning", "etype": "process", "name": "Nickel electrowinning"},
			{"slug": "parameter:temperature", "etype": "parameter", "name": "temperature"},
		},
		"numeric_facts": []map[string]any{
			{"subject_slug": "process:nickel-electrowinning", "parameter_slug": "parameter:temperature", "operator": "range", "vmin": 60, "vmax": 80, "unit_orig": "C", "unit_code": "degC", "quote": "60-80 C", "confidence": 0.9},
		},
	})

	_, err := service.CommitExtraction(t.Context(), bundleURI)
	if err != nil {
		t.Fatalf("commit extraction: %v", err)
	}

	if repo.committed.Version != 1 {
		t.Fatalf("expected default bundle version 1, got %d", repo.committed.Version)
	}
	if len(repo.committed.Facts) != 1 {
		t.Fatalf("expected one legacy fact, got %d", len(repo.committed.Facts))
	}
	if repo.committed.Facts[0].Relation != "operates_at" {
		t.Fatalf("expected default relation operates_at, got %q", repo.committed.Facts[0].Relation)
	}
	if repo.committed.Facts[0].ExtractorVersion != "legacy-extract" {
		t.Fatalf("expected bundle extractor version, got %q", repo.committed.Facts[0].ExtractorVersion)
	}
}

func serviceWithBundle(t *testing.T, repo *fakeRepository, payload map[string]any) (*Service, string) {
	t.Helper()
	store := blob.NewMemStore()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal bundle: %v", err)
	}
	uri, err := store.Put(t.Context(), "bundles", "bundle.json", bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatalf("put bundle: %v", err)
	}
	return New(repo, store), uri
}
