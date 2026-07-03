package app

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"github.com/maksim-mshp/nornickel-hackathon/internal/ingest/domain"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/events"
)

const eventSource = "kmap/ingest"

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

type RegisterCommand struct {
	Title       string
	BlobURI     string
	SHA256      []byte
	DocType     string
	Lang        string
	Geography   string
	AccessLevel string
	Year        int
	UploadedBy  string
	Meta        map[string]any
}

type RegisterResult struct {
	DocumentID uuid.UUID
	Version    int
	Status     string
	Duplicate  bool
}

func (service *Service) RegisterDocument(ctx context.Context, cmd RegisterCommand) (RegisterResult, error) {
	if len(cmd.SHA256) == 0 {
		return RegisterResult{}, domain.ErrSHA256Required
	}
	if cmd.BlobURI == "" {
		return RegisterResult{}, domain.ErrBlobURIRequired
	}

	meta, err := domain.NormalizeMeta(cmd.DocType, cmd.Geography, cmd.AccessLevel)
	if err != nil {
		return RegisterResult{}, err
	}

	if existingID, found, err := service.repository.FindIDBySHA256(ctx, cmd.SHA256); err != nil {
		return RegisterResult{}, fmt.Errorf("dedup lookup: %w", err)
	} else if found {
		return RegisterResult{DocumentID: existingID, Version: 1, Status: domain.StatusRegistered, Duplicate: true}, nil
	}

	id, err := uuid.NewV7()
	if err != nil {
		return RegisterResult{}, fmt.Errorf("generate document id: %w", err)
	}

	doc := domain.Document{
		ID:          id,
		Title:       cmd.Title,
		DocType:     meta.DocType,
		Lang:        cmd.Lang,
		Year:        cmd.Year,
		Geography:   meta.Geography,
		AccessLevel: meta.AccessLevel,
		SourceURI:   cmd.BlobURI,
		SHA256:      cmd.SHA256,
		Status:      domain.StatusRegistered,
		Version:     1,
		UploadedBy:  cmd.UploadedBy,
		Meta:        cmd.Meta,
		BlobURI:     cmd.BlobURI,
	}

	envelope, err := service.buildEnvelope(doc)
	if err != nil {
		return RegisterResult{}, err
	}

	saved, err := service.repository.Register(ctx, doc, envelope)
	if err != nil {
		return RegisterResult{}, err
	}
	return RegisterResult{DocumentID: saved.ID, Version: saved.Version, Status: saved.Status}, nil
}

type StatusResult struct {
	Document domain.Document
	Stages   []domain.Stage
}

func (service *Service) GetStatus(ctx context.Context, documentID uuid.UUID) (StatusResult, error) {
	doc, stages, err := service.repository.GetStatus(ctx, documentID)
	if err != nil {
		return StatusResult{}, err
	}
	return StatusResult{Document: doc, Stages: stages}, nil
}

func (service *Service) ListDocuments(ctx context.Context, cursor string, limit uint32) ([]DocumentSummary, string, error) {
	if limit == 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	return service.repository.ListDocuments(ctx, cursor, limit)
}

func (service *Service) buildEnvelope(doc domain.Document) (events.Envelope, error) {
	payload := map[string]any{
		"document_id": doc.ID.String(),
		"version":     doc.Version,
		"sha256":      hex.EncodeToString(doc.SHA256),
		"blob_uri":    doc.BlobURI,
		"declared_meta": map[string]any{
			"doc_type":     doc.DocType,
			"lang":         doc.Lang,
			"geography":    doc.Geography,
			"access_level": doc.AccessLevel,
		},
	}
	return events.New(events.Event{
		Type:    events.DocumentRegistered,
		Source:  eventSource,
		Subject: doc.ID.String(),
		Data:    payload,
	})
}
