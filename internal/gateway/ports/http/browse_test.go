package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

type fakeSearchClient struct {
	experts *kmapv1.ListExpertsResponse
	graph   *kmapv1.EgoGraphResponse
}

func (client fakeSearchClient) Search(context.Context, *kmapv1.SearchRequest, ...grpc.CallOption) (*kmapv1.SearchResponse, error) {
	return &kmapv1.SearchResponse{}, nil
}

func (client fakeSearchClient) EgoGraph(context.Context, *kmapv1.EgoGraphRequest, ...grpc.CallOption) (*kmapv1.EgoGraphResponse, error) {
	return client.graph, nil
}

func (client fakeSearchClient) ListExperts(context.Context, *kmapv1.ListExpertsRequest, ...grpc.CallOption) (*kmapv1.ListExpertsResponse, error) {
	return client.experts, nil
}

func (client fakeSearchClient) ListEntities(context.Context, *kmapv1.ListEntitiesRequest, ...grpc.CallOption) (*kmapv1.ListEntitiesResponse, error) {
	return &kmapv1.ListEntitiesResponse{}, nil
}

func (client fakeSearchClient) GetEntity(context.Context, *kmapv1.GetEntityRequest, ...grpc.CallOption) (*kmapv1.GetEntityResponse, error) {
	return &kmapv1.GetEntityResponse{}, nil
}

func (client fakeSearchClient) ListEntityFacts(context.Context, *kmapv1.ListEntityFactsRequest, ...grpc.CallOption) (*kmapv1.ListEntityFactsResponse, error) {
	return &kmapv1.ListEntityFactsResponse{}, nil
}

func (client fakeSearchClient) ListExperiments(context.Context, *kmapv1.ListExperimentsRequest, ...grpc.CallOption) (*kmapv1.ListExperimentsResponse, error) {
	return &kmapv1.ListExperimentsResponse{}, nil
}

type fakeEpistemicClient struct {
	coverage *kmapv1.GetCoverageResponse
}

func (client fakeEpistemicClient) GetCoverage(context.Context, *kmapv1.GetCoverageRequest, ...grpc.CallOption) (*kmapv1.GetCoverageResponse, error) {
	return client.coverage, nil
}

func (client fakeEpistemicClient) GetContradictions(context.Context, *kmapv1.GetContradictionsRequest, ...grpc.CallOption) (*kmapv1.GetContradictionsResponse, error) {
	return &kmapv1.GetContradictionsResponse{}, nil
}

func (client fakeEpistemicClient) DecideContradiction(context.Context, *kmapv1.DecideContradictionRequest, ...grpc.CallOption) (*kmapv1.DecideContradictionResponse, error) {
	return &kmapv1.DecideContradictionResponse{}, nil
}

type fakeCatalogClient struct{}

func (client fakeCatalogClient) CommitExtraction(context.Context, *kmapv1.CommitExtractionRequest, ...grpc.CallOption) (*kmapv1.CommitExtractionResponse, error) {
	return &kmapv1.CommitExtractionResponse{}, nil
}

func (client fakeCatalogClient) ResolveEntities(context.Context, *kmapv1.ResolveEntitiesRequest, ...grpc.CallOption) (*kmapv1.ResolveEntitiesResponse, error) {
	return &kmapv1.ResolveEntitiesResponse{}, nil
}

func (client fakeCatalogClient) MergeEntities(context.Context, *kmapv1.MergeEntitiesRequest, ...grpc.CallOption) (*kmapv1.MergeEntitiesResponse, error) {
	return &kmapv1.MergeEntitiesResponse{}, nil
}

func (client fakeCatalogClient) UpdateFactStatus(context.Context, *kmapv1.UpdateFactStatusRequest, ...grpc.CallOption) (*kmapv1.UpdateFactStatusResponse, error) {
	return &kmapv1.UpdateFactStatusResponse{}, nil
}

func (client fakeCatalogClient) UpsertSeed(context.Context, *kmapv1.UpsertSeedRequest, ...grpc.CallOption) (*kmapv1.UpsertSeedResponse, error) {
	return &kmapv1.UpsertSeedResponse{}, nil
}

func TestExpertsHandlerMapsFrontendDTO(t *testing.T) {
	t.Parallel()

	evidence := mustStruct(t, map[string]any{
		"lab":         "Лаборатория",
		"reports":     float64(3),
		"experiments": float64(2),
		"lastYear":    float64(2025),
		"topics":      []any{"электроэкстракция"},
	})
	server := &Server{search: fakeSearchClient{experts: &kmapv1.ListExpertsResponse{
		Experts: []*kmapv1.Expert{{PersonId: "person:ivanov", Name: "Иванов", Weight: 0.83, Evidence: evidence}},
		Page:    &kmapv1.PageResponse{},
	}}}
	req := httptest.NewRequest(http.MethodGet, "/v1/experts?entity_id=process:x", nil)
	rec := httptest.NewRecorder()

	server.expertsHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected application/json, got %q", rec.Header().Get("Content-Type"))
	}
	for _, expected := range []string{`"id":"person:ivanov"`, `"lab":"Лаборатория"`, `"lastYear":2025`} {
		if !strings.Contains(rec.Body.String(), expected) {
			t.Fatalf("expected response to contain %s, got %s", expected, rec.Body.String())
		}
	}
}

func TestCoverageHandlerMapsCells(t *testing.T) {
	t.Parallel()

	server := &Server{epistemic: fakeEpistemicClient{coverage: &kmapv1.GetCoverageResponse{
		Cells: []*kmapv1.CoverageCell{{
			Id: "cell-1", Domain: "hydrometallurgy", MaterialId: "материал", ProcessId: "процесс",
			ConditionKey: "base", Score: 0.7, GapFlag: true, GapReasons: []string{"нет данных"},
			Counters: mustStruct(t, map[string]any{"material": "католит", "process": "электроэкстракция"}),
		}},
	}}}
	req := httptest.NewRequest(http.MethodGet, "/v1/coverage?domain=hydrometallurgy", nil)
	rec := httptest.NewRecorder()

	server.coverageHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"gap_flag":true`) {
		t.Fatalf("expected gap flag in response, got %s", rec.Body.String())
	}
}

func TestUpdateFactStatusRequiresStatus(t *testing.T) {
	t.Parallel()

	server := &Server{catalog: fakeCatalogClient{}}
	req := httptest.NewRequest(http.MethodPost, "/v1/facts/f1/status", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	server.updateFactStatusHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func mustStruct(t *testing.T, values map[string]any) *structpb.Struct {
	t.Helper()
	result, err := structpb.NewStruct(values)
	if err != nil {
		t.Fatalf("build struct: %v", err)
	}
	return result
}
