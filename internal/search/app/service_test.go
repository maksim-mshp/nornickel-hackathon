package app

import (
	"context"
	"testing"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

type fakeRepo struct {
	facts          []Fact
	consensus      []Consensus
	contradictions []Contradiction
	gaps           []GapCell
	experts        []Expert
}

func (f fakeRepo) ExpandEntityIDs(_ context.Context, slugs []string) ([]string, error) {
	return slugs, nil
}
func (f fakeRepo) Facts(context.Context, []string, FactFilter) ([]Fact, error) { return f.facts, nil }
func (f fakeRepo) Consensus(context.Context, []string) ([]Consensus, error) { return f.consensus, nil }
func (f fakeRepo) Contradictions(context.Context, []string) ([]Contradiction, error) {
	return f.contradictions, nil
}
func (f fakeRepo) Gaps(context.Context, []string) ([]GapCell, error)   { return f.gaps, nil }
func (f fakeRepo) Experts(context.Context, []string) ([]Expert, error) { return f.experts, nil }
func (f fakeRepo) EgoGraph(context.Context, []string) ([]GraphNode, []GraphEdge, error) {
	return []GraphNode{{ID: "n1", Type: "process", Label: "p"}}, nil, nil
}
func (f fakeRepo) ListEntities(context.Context, string, string, uint32) ([]*kmapv1.EntitySummary, error) {
	return nil, nil
}
func (f fakeRepo) GetEntity(context.Context, string) (*kmapv1.EntityCard, error) {
	return &kmapv1.EntityCard{}, nil
}
func (f fakeRepo) ListExperiments(context.Context, *kmapv1.ListExperimentsRequest) ([]*kmapv1.ExperimentSummary, error) {
	return nil, nil
}

func sampleFacts() []Fact {
	return []Fact{
		{
			ID: "id-weak", Subject: EntityRef{Slug: "process:x", Name: "X"}, Parameter: EntityRef{Slug: "parameter:z", Name: "Z"},
			Value:      NumericValue{Operator: "eq", Vmin: ptr(1), Vmax: ptr(1), Unit: "м/с"},
			Provenance: Provenance{DocumentID: "d1", DocType: "web", Year: 2015}, Confidence: 0.7,
			ValidationStatus: "machine_extracted", Geography: "foreign",
		},
		{
			ID: "id-strong", Subject: EntityRef{Slug: "process:x", Name: "X"}, Parameter: EntityRef{Slug: "parameter:y", Name: "Y"},
			Value:      NumericValue{Operator: "range", Vmin: ptr(2), Vmax: ptr(3), Unit: "м/с"},
			Provenance: Provenance{DocumentID: "d2", DocType: "report", Year: 2025}, Confidence: 0.99,
			ValidationStatus: "expert_validated", Geography: "ru",
		},
	}
}

func ptr(v float64) *float64 { return &v }

func TestSearchRanksAndAssignsRefs(t *testing.T) {
	t.Parallel()

	repo := fakeRepo{
		facts: sampleFacts(),
		contradictions: []Contradiction{
			{ID: "c1", AFactRef: "id-strong", BFactRef: "id-weak", Cause: "reason", Status: "judge_confirmed", Confidence: 0.8},
		},
	}
	service := NewService(repo, DefaultRanking(), 2026)

	plan := &kmapv1.QueryPlan{Entities: mustStruct(t, map[string]any{
		"processes": []any{map[string]any{"slug": "process:x", "name": "X"}},
	})}
	resp, err := service.Search(context.Background(), &kmapv1.SearchRequest{Plan: plan})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	facts := resp.GetEvidence().GetFacts()
	if len(facts) != 2 {
		t.Fatalf("expected 2 facts, got %d", len(facts))
	}
	first := facts[0].GetPayload().GetFields()
	if first["ref"].GetStringValue() != "F1" {
		t.Fatalf("expected first ref F1, got %s", first["ref"].GetStringValue())
	}
	if first["id"].GetStringValue() != "id-strong" {
		t.Fatalf("expected strongest fact first, got %s", first["id"].GetStringValue())
	}

	contradiction := resp.GetEvidence().GetContradictions()[0].GetFields()
	if contradiction["aFactRef"].GetStringValue() != "F1" || contradiction["bFactRef"].GetStringValue() != "F2" {
		t.Fatalf("contradiction refs not remapped: %v/%v", contradiction["aFactRef"], contradiction["bFactRef"])
	}

	stats := resp.GetEvidence().GetStats().GetFields()
	if stats["sources"].GetNumberValue() != 2 {
		t.Fatalf("expected 2 sources, got %v", stats["sources"].GetNumberValue())
	}
	if stats["ruSources"].GetNumberValue() != 1 || stats["foreignSources"].GetNumberValue() != 1 {
		t.Fatalf("unexpected geography split: %v", stats)
	}
}

func mustStruct(t *testing.T, value map[string]any) *structpb.Struct {
	t.Helper()
	result, err := structpb.NewStruct(value)
	if err != nil {
		t.Fatalf("build struct: %v", err)
	}
	return result
}
