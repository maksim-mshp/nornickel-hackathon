package app

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/maksim-mshp/nornickel-hackathon/internal/ingest/domain"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/events"
)

type fakeRepository struct {
	docs        map[uuid.UUID]domain.Document
	stages      map[uuid.UUID][]domain.Stage
	registerErr error
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		docs:   map[uuid.UUID]domain.Document{},
		stages: map[uuid.UUID][]domain.Stage{},
	}
}

func (repo *fakeRepository) FindIDBySHA256(_ context.Context, sha256 []byte) (uuid.UUID, bool, error) {
	for _, doc := range repo.docs {
		if slices.Equal(doc.SHA256, sha256) {
			return doc.ID, true, nil
		}
	}
	return uuid.Nil, false, nil
}

func (repo *fakeRepository) Register(_ context.Context, doc domain.Document, _ events.Envelope) (domain.Document, error) {
	if repo.registerErr != nil {
		return domain.Document{}, repo.registerErr
	}
	repo.docs[doc.ID] = doc
	repo.stages[doc.ID] = domain.DefaultStages()
	return doc, nil
}

func (repo *fakeRepository) GetStatus(_ context.Context, documentID uuid.UUID) (domain.Document, []domain.Stage, error) {
	doc, ok := repo.docs[documentID]
	if !ok {
		return domain.Document{}, nil, domain.ErrDocumentNotFound
	}
	return doc, repo.stages[documentID], nil
}

func TestRegisterDocumentRequiresSHA256(t *testing.T) {
	t.Parallel()

	service := NewService(newFakeRepository())
	_, err := service.RegisterDocument(t.Context(), RegisterCommand{BlobURI: "s3://kmap-raw/x"})
	if !errors.Is(err, domain.ErrSHA256Required) {
		t.Fatalf("expected ErrSHA256Required, got %v", err)
	}
}

func TestRegisterDocumentRequiresBlobURI(t *testing.T) {
	t.Parallel()

	service := NewService(newFakeRepository())
	_, err := service.RegisterDocument(t.Context(), RegisterCommand{SHA256: []byte("h")})
	if !errors.Is(err, domain.ErrBlobURIRequired) {
		t.Fatalf("expected ErrBlobURIRequired, got %v", err)
	}
}

func TestRegisterDocumentRejectsInvalidDocType(t *testing.T) {
	t.Parallel()

	service := NewService(newFakeRepository())
	_, err := service.RegisterDocument(t.Context(), RegisterCommand{
		BlobURI: "s3://kmap-raw/x",
		SHA256:  []byte("h"),
		DocType: "bogus",
	})
	if !errors.Is(err, domain.ErrInvalidDocType) {
		t.Fatalf("expected ErrInvalidDocType, got %v", err)
	}
}

func TestRegisterDocumentAppliesDefaults(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	service := NewService(repo)

	result, err := service.RegisterDocument(t.Context(), RegisterCommand{
		Title:   "report",
		BlobURI: "s3://kmap-raw/x",
		SHA256:  []byte("h"),
	})
	if err != nil {
		t.Fatalf("expected register to succeed: %v", err)
	}
	if result.Duplicate {
		t.Fatal("expected non-duplicate registration")
	}

	doc := repo.docs[result.DocumentID]
	if doc.DocType != "report" || doc.Geography != "unknown" || doc.AccessLevel != "internal" {
		t.Fatalf("unexpected defaults: %+v", doc)
	}
	if doc.Status != domain.StatusRegistered || doc.Version != 1 {
		t.Fatalf("unexpected status/version: %s/%d", doc.Status, doc.Version)
	}
}

func TestRegisterDocumentDeduplicatesBySHA256(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	service := NewService(repo)

	first, err := service.RegisterDocument(t.Context(), RegisterCommand{BlobURI: "s3://kmap-raw/x", SHA256: []byte("dup")})
	if err != nil {
		t.Fatalf("first register: %v", err)
	}
	second, err := service.RegisterDocument(t.Context(), RegisterCommand{BlobURI: "s3://kmap-raw/y", SHA256: []byte("dup")})
	if err != nil {
		t.Fatalf("second register: %v", err)
	}
	if second.DocumentID != first.DocumentID || !second.Duplicate {
		t.Fatalf("expected duplicate of %s, got %+v", first.DocumentID, second)
	}
	if len(repo.docs) != 1 {
		t.Fatalf("expected 1 stored document, got %d", len(repo.docs))
	}
}

func TestGetStatusReturnsStages(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	service := NewService(repo)

	registered, err := service.RegisterDocument(t.Context(), RegisterCommand{BlobURI: "s3://kmap-raw/x", SHA256: []byte("h")})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	result, err := service.GetStatus(t.Context(), registered.DocumentID)
	if err != nil {
		t.Fatalf("get status: %v", err)
	}
	if len(result.Stages) != len(domain.DefaultStages()) {
		t.Fatalf("expected %d stages, got %d", len(domain.DefaultStages()), len(result.Stages))
	}
}

func TestGetStatusNotFound(t *testing.T) {
	t.Parallel()

	service := NewService(newFakeRepository())
	_, err := service.GetStatus(t.Context(), uuid.New())
	if !errors.Is(err, domain.ErrDocumentNotFound) {
		t.Fatalf("expected ErrDocumentNotFound, got %v", err)
	}
}
