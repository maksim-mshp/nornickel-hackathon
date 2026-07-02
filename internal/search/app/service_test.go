package app

import (
	"context"
	"testing"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
)

func TestSearchReturnsEmptyEvidencePack(t *testing.T) {
	t.Parallel()

	service := NewService()
	resp, err := service.Search(context.Background(), &kmapv1.SearchRequest{})
	if err != nil {
		t.Fatalf("expected search to succeed: %v", err)
	}
	if resp.GetEvidence() == nil {
		t.Fatal("expected evidence pack")
	}
	if resp.GetEvidence().GetGraph() == nil {
		t.Fatal("expected graph")
	}
}

func TestEgoGraphReturnsGraph(t *testing.T) {
	t.Parallel()

	service := NewService()
	resp, err := service.EgoGraph(context.Background(), &kmapv1.EgoGraphRequest{})
	if err != nil {
		t.Fatalf("expected ego graph to succeed: %v", err)
	}
	if resp.GetGraph() == nil {
		t.Fatal("expected graph")
	}
}

func TestListExpertsReturnsPage(t *testing.T) {
	t.Parallel()

	service := NewService()
	resp, err := service.ListExperts(context.Background(), &kmapv1.ListExpertsRequest{})
	if err != nil {
		t.Fatalf("expected list experts to succeed: %v", err)
	}
	if resp.GetPage() == nil {
		t.Fatal("expected page")
	}
}
