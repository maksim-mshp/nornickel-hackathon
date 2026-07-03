package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/blob"
)

type Repo interface {
	ResolveEntities(ctx context.Context, names []string) ([]Resolution, error)
	CommitExtraction(ctx context.Context, bundle Bundle) (CommitResult, error)
	UpdateFactStatus(ctx context.Context, factID string, factKind string, status string, actor string, comment string) error
	MergeEntities(ctx context.Context, entityID string, intoID string, actor string, comment string) error
}

type Service struct {
	repo Repo
	blob blob.Store
}

func NewService(repo Repo, store blob.Store) *Service {
	return &Service{repo: repo, blob: store}
}

func (service *Service) ResolveEntities(ctx context.Context, names []string) ([]Resolution, error) {
	return service.repo.ResolveEntities(ctx, names)
}

func (service *Service) CommitExtraction(ctx context.Context, bundleURI string) (CommitResult, error) {
	bundle, err := service.loadBundle(ctx, bundleURI)
	if err != nil {
		return CommitResult{}, err
	}
	return service.repo.CommitExtraction(ctx, bundle)
}

func (service *Service) UpdateFactStatus(ctx context.Context, factID string, factKind string, status string, actor string, comment string) error {
	return service.repo.UpdateFactStatus(ctx, factID, factKind, status, actor, comment)
}

func (service *Service) MergeEntities(ctx context.Context, entityID string, intoID string, actor string, comment string) error {
	return service.repo.MergeEntities(ctx, entityID, intoID, actor, comment)
}

func (service *Service) loadBundle(ctx context.Context, bundleURI string) (Bundle, error) {
	bucket, key, err := blob.ParseURI(bundleURI)
	if err != nil {
		return Bundle{}, err
	}
	reader, err := service.blob.Get(ctx, bucket, key)
	if err != nil {
		return Bundle{}, err
	}
	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(reader)
	if err != nil {
		return Bundle{}, fmt.Errorf("read bundle: %w", err)
	}
	var bundle Bundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return Bundle{}, fmt.Errorf("decode bundle: %w", err)
	}
	return bundle, nil
}
