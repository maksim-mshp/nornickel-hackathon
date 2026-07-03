package app

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"strings"

	"github.com/google/uuid"
	"github.com/maksim-mshp/nornickel-hackathon/internal/catalog/domain"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/blob"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/events"
)

const eventSource = "kmap/catalog"

type Service struct {
	repository Repository
	blobs      blob.Store
}

func New(repository Repository, blobs blob.Store) *Service {
	return &Service{repository: repository, blobs: blobs}
}

func (service *Service) UpdateFactStatus(ctx context.Context, factID string, factKind string, status string, actor string, comment string) error {
	return service.repository.UpdateFactStatus(ctx, factID, factKind, status, actor, comment)
}

func (service *Service) MergeEntities(ctx context.Context, entityID string, intoID string, actor string, comment string) error {
	return service.repository.MergeEntities(ctx, entityID, intoID, actor, comment)
}

type CommitResult struct {
	DocumentID  uuid.UUID
	FactIDs     []uuid.UUID
	EntityIDs   []uuid.UUID
	ClusterKeys []string
}

func (service *Service) CommitExtraction(ctx context.Context, bundleURI string) (CommitResult, error) {
	if bundleURI == "" {
		return CommitResult{}, domain.ErrBundleURIRequired
	}
	bundle, err := service.fetchBundle(ctx, bundleURI)
	if err != nil {
		return CommitResult{}, err
	}
	documentID, err := uuid.Parse(bundle.DocumentID)
	if err != nil {
		return CommitResult{}, fmt.Errorf("%w: invalid document_id", domain.ErrInvalidBundle)
	}

	cmd, err := service.buildCommand(ctx, bundle, documentID)
	if err != nil {
		return CommitResult{}, err
	}

	committed, clusterDirty, err := service.buildEnvelopes(cmd)
	if err != nil {
		return CommitResult{}, err
	}
	if err := service.repository.Commit(ctx, cmd, committed, clusterDirty); err != nil {
		return CommitResult{}, err
	}

	return CommitResult{
		DocumentID:  documentID,
		FactIDs:     factIDs(cmd.Facts),
		EntityIDs:   entityIDs(cmd),
		ClusterKeys: clusterKeys(cmd.Facts),
	}, nil
}

type ResolveResult struct {
	Input      string
	EntityID   string
	Slug       string
	Name       string
	Confidence float64
	Status     string
}

func (service *Service) ResolveEntities(ctx context.Context, names []string) ([]ResolveResult, error) {
	resolved, err := service.repository.ResolveByNames(ctx, names)
	if err != nil {
		return nil, err
	}
	results := make([]ResolveResult, 0, len(names))
	for _, name := range names {
		result := ResolveResult{Input: name, Status: "new"}
		if id, ok := resolved[normalizeName(name)]; ok {
			result.EntityID = id.String()
			result.Slug = ""
			result.Confidence = 1.0
			result.Status = "resolved"
		}
		results = append(results, result)
	}
	return results, nil
}

func (service *Service) fetchBundle(ctx context.Context, bundleURI string) (domain.Bundle, error) {
	bucket, key, err := blob.ParseURI(bundleURI)
	if err != nil {
		return domain.Bundle{}, fmt.Errorf("parse bundle uri: %w", err)
	}
	reader, err := service.blobs.Get(ctx, bucket, key)
	if err != nil {
		return domain.Bundle{}, fmt.Errorf("fetch bundle: %w", err)
	}
	defer func() { _ = reader.Close() }()
	data, err := io.ReadAll(reader)
	if err != nil {
		return domain.Bundle{}, fmt.Errorf("read bundle: %w", err)
	}
	var bundle domain.Bundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return domain.Bundle{}, fmt.Errorf("%w: %v", domain.ErrInvalidBundle, err)
	}
	if bundle.DocumentID == "" {
		return domain.Bundle{}, fmt.Errorf("%w: missing document_id", domain.ErrInvalidBundle)
	}
	return bundle, nil
}

func (service *Service) buildCommand(ctx context.Context, bundle domain.Bundle, documentID uuid.UUID) (CommitCommand, error) {
	names := uniqueNames(bundle)
	resolved, err := service.repository.ResolveByNames(ctx, names)
	if err != nil {
		return CommitCommand{}, fmt.Errorf("resolve entities: %w", err)
	}

	nameIndex := maps.Clone(resolved)
	var newEntities []domain.Entity

	resolveRef := func(name string, defaultEtype string) (uuid.UUID, error) {
		key := normalizeName(name)
		if id, ok := nameIndex[key]; ok {
			return id, nil
		}
		etype, canonical := entityTypeAndName(defaultEtype, name)
		entity, err := domain.NewEntity(etype, canonical, "")
		if err != nil {
			return uuid.Nil, err
		}
		nameIndex[key] = entity.ID
		nameIndex[normalizeName(entity.Slug)] = entity.ID
		newEntities = append(newEntities, entity)
		return entity.ID, nil
	}

	for _, item := range bundle.Entities {
		entityName := defaultString(item.Name, item.Slug)
		if _, _, ok := lookup(nameIndex, entityName); ok {
			continue
		}
		entityType := defaultString(item.Type, item.EType)
		if _, err := resolveRef(entityName, entityType); err != nil {
			return CommitCommand{}, err
		}
	}

	chunkIDs := map[string]uuid.UUID{}
	var chunks []ChunkInsert
	for _, chunk := range bundle.Chunks {
		id, err := uuid.NewV7()
		if err != nil {
			return CommitCommand{}, fmt.Errorf("generate chunk id: %w", err)
		}
		chunkIDs[chunk.ID] = id
		chunks = append(chunks, ChunkInsert{UUID: id, Chunk: chunk})
	}

	var facts []domain.NumericFact
	for _, candidate := range numericCandidates(bundle) {
		if err := domain.ValidateOperator(candidate.Operator); err != nil {
			return CommitCommand{}, err
		}
		subjectID, err := resolveRef(defaultString(candidate.Subject, candidate.SubjectSlug), "process")
		if err != nil {
			return CommitCommand{}, err
		}
		parameterID, err := resolveRef(defaultString(candidate.Parameter, candidate.ParameterSlug), "parameter")
		if err != nil {
			return CommitCommand{}, err
		}
		factID, err := uuid.NewV7()
		if err != nil {
			return CommitCommand{}, fmt.Errorf("generate fact id: %w", err)
		}
		fact := domain.NumericFact{
			ID:               factID,
			SubjectID:        subjectID,
			ParameterID:      parameterID,
			Operator:         candidate.Operator,
			ValueRaw:         candidate.ValueRaw,
			VMin:             candidate.VMin,
			VMax:             candidate.VMax,
			UnitOrig:         candidate.UnitOrig,
			UnitCode:         candidate.UnitCode,
			VMinSI:           candidate.VMinSI,
			VMaxSI:           candidate.VMaxSI,
			Conditions:       candidate.Conditions,
			ConditionHash:    candidate.ConditionHash,
			Quote:            candidate.Quote,
			Page:             candidate.Page,
			CharFrom:         candidate.CharFrom,
			CharTo:           candidate.CharTo,
			Geography:        candidate.Geography,
			ExtractionMethod: defaultString(candidate.ExtractionMethod, domain.MethodDeterministic),
			ExtractorVersion: defaultString(candidate.ExtractorVersion, bundle.ExtractorVersion),
			Confidence:       candidate.Confidence,
		}
		if candidate.Relation != "" {
			fact.Relation = candidate.Relation
		} else {
			fact.Relation = "operates_at"
		}
		if chunkID, ok := chunkIDs[candidate.ChunkID]; ok {
			fact.ChunkID = &chunkID
		}
		facts = append(facts, fact)
	}

	return CommitCommand{
		DocumentID:  documentID,
		Version:     bundleVersion(bundle.Version),
		NewEntities: newEntities,
		Chunks:      chunks,
		Facts:       facts,
	}, nil
}

func (service *Service) buildEnvelopes(cmd CommitCommand) (events.Envelope, events.Envelope, error) {
	committed, err := events.New(events.Event{
		Type:    events.FactsCommitted,
		Source:  eventSource,
		Subject: cmd.DocumentID.String(),
		Data: map[string]any{
			"document_id":  cmd.DocumentID.String(),
			"version":      cmd.Version,
			"fact_ids":     idStrings(cmd.FactIDs()),
			"entity_ids":   idStrings(cmd.EntityIDs()),
			"cluster_keys": clusterKeys(cmd.Facts),
		},
	})
	if err != nil {
		return events.Envelope{}, events.Envelope{}, err
	}
	clusterDirty, err := events.New(events.Event{
		Type:    events.EpistemicClusterDirty,
		Source:  eventSource,
		Subject: cmd.DocumentID.String(),
		Data: map[string]any{
			"document_id":  cmd.DocumentID.String(),
			"cluster_keys": clusterKeys(cmd.Facts),
		},
	})
	if err != nil {
		return events.Envelope{}, events.Envelope{}, err
	}
	return committed, clusterDirty, nil
}

func uniqueNames(bundle domain.Bundle) []string {
	seen := map[string]struct{}{}
	add := func(name string) {
		name = normalizeName(name)
		if name == "" {
			return
		}
		seen[name] = struct{}{}
	}
	for _, entity := range bundle.Entities {
		add(entity.Name)
		add(entity.Slug)
	}
	for _, candidate := range numericCandidates(bundle) {
		add(candidate.Subject)
		add(candidate.SubjectSlug)
		add(candidate.Parameter)
		add(candidate.ParameterSlug)
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	return names
}

func lookup(index map[string]uuid.UUID, name string) (uuid.UUID, string, bool) {
	key := normalizeName(name)
	id, ok := index[key]
	return id, key, ok
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func factIDs(facts []domain.NumericFact) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(facts))
	for _, fact := range facts {
		ids = append(ids, fact.ID)
	}
	return ids
}

func (cmd CommitCommand) FactIDs() []uuid.UUID {
	return factIDs(cmd.Facts)
}

func (cmd CommitCommand) EntityIDs() []uuid.UUID {
	seen := map[uuid.UUID]struct{}{}
	ids := make([]uuid.UUID, 0)
	for _, entity := range cmd.NewEntities {
		if _, ok := seen[entity.ID]; ok {
			continue
		}
		seen[entity.ID] = struct{}{}
		ids = append(ids, entity.ID)
	}
	for _, fact := range cmd.Facts {
		for _, id := range []uuid.UUID{fact.SubjectID, fact.ParameterID} {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	return ids
}

func entityIDs(cmd CommitCommand) []uuid.UUID {
	return cmd.EntityIDs()
}

func clusterKeys(facts []domain.NumericFact) []string {
	seen := map[string]struct{}{}
	keys := make([]string, 0, len(facts))
	for _, fact := range facts {
		key := fmt.Sprintf("%s:%s:%s", fact.SubjectID, fact.ParameterID, hex.EncodeToString(fact.ConditionHash))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	return keys
}

func idStrings(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}

func numericCandidates(bundle domain.Bundle) []domain.NumericCandidate {
	if len(bundle.NumericCandidates) > 0 {
		return bundle.NumericCandidates
	}
	return bundle.NumericFacts
}

func entityTypeAndName(defaultEtype string, ref string) (string, string) {
	if before, after, ok := strings.Cut(ref, ":"); ok && before != "" && after != "" {
		return before, strings.ReplaceAll(after, "-", " ")
	}
	return defaultEtype, ref
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func bundleVersion(version int) int {
	if version == 0 {
		return 1
	}
	return version
}
