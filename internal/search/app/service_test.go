package app

import (
	"context"
	"testing"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestSearchReturnsFactsWithProvenance(t *testing.T) {
	t.Parallel()

	service := NewService()
	resp, err := service.Search(context.Background(), &kmapv1.SearchRequest{})
	if err != nil {
		t.Fatalf("expected search to succeed: %v", err)
	}
	pack := resp.GetEvidence()
	if pack == nil {
		t.Fatal("expected evidence pack")
	}
	if len(pack.GetFacts()) == 0 {
		t.Fatal("expected facts")
	}
	first := pack.GetFacts()[0].GetPayload()
	if first == nil {
		t.Fatal("expected fact payload")
	}
	prov := first.GetFields()["provenance"].GetStructValue()
	if prov.GetFields()["quote"].GetStringValue() == "" {
		t.Fatal("expected fact provenance quote")
	}
	if pack.GetStats().GetFields()["sources"].GetNumberValue() == 0 {
		t.Fatal("expected non-zero source stats")
	}
}

func TestSearchSelectsScenarioByPlanSlug(t *testing.T) {
	t.Parallel()

	plan := &kmapv1.QueryPlan{
		Entities: mustStruct(t, map[string]any{
			"processes": []any{map[string]any{"slug": "process:desalination", "name": "обессоливание"}},
		}),
	}

	service := NewService()
	resp, err := service.Search(context.Background(), &kmapv1.SearchRequest{Plan: plan})
	if err != nil {
		t.Fatalf("expected search to succeed: %v", err)
	}
	subject := resp.GetEvidence().GetFacts()[0].GetPayload().GetFields()["subject"].GetStructValue()
	if got := subject.GetFields()["slug"].GetStringValue(); got != "technology:reverse-osmosis" {
		t.Fatalf("expected desalination scenario, got subject %q", got)
	}
}

func TestEgoGraphReturnsGraph(t *testing.T) {
	t.Parallel()

	service := NewService()
	resp, err := service.EgoGraph(context.Background(), &kmapv1.EgoGraphRequest{})
	if err != nil {
		t.Fatalf("expected ego graph to succeed: %v", err)
	}
	if len(resp.GetGraph().GetNodes()) == 0 {
		t.Fatal("expected graph nodes")
	}
}

func TestListExpertsReturnsExperts(t *testing.T) {
	t.Parallel()

	service := NewService()
	resp, err := service.ListExperts(context.Background(), &kmapv1.ListExpertsRequest{})
	if err != nil {
		t.Fatalf("expected list experts to succeed: %v", err)
	}
	if len(resp.GetExperts()) == 0 {
		t.Fatal("expected experts")
	}
	if resp.GetPage() == nil {
		t.Fatal("expected page")
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
