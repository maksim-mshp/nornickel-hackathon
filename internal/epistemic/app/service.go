package app

import (
	"context"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type CoverageCell struct {
	ID              string
	Domain          string
	MaterialName    string
	ProcessName     string
	ConditionKey    string
	Score           float64
	GapFlag         bool
	GapReasons      []string
	Docs            int
	Experiments     int
	Facts           int
	Experts         int
	RuDocs          int
	ForeignDocs     int
	ValidatedFacts  int
	LastSourceYear  int
	ScoreComponents map[string]float64
}

type Contradiction struct {
	ID          string
	ClusterID   string
	Status      string
	Dtype       string
	Severity    float64
	Subject     string
	Parameter   string
	AStatement  string
	BStatement  string
	Cause       string
	Confounders []string
}

type Repo interface {
	Coverage(ctx context.Context, domain string) ([]CoverageCell, error)
	Contradictions(ctx context.Context, status string, entityID string) ([]Contradiction, error)
	DecideContradiction(ctx context.Context, id string, status string, rationale string) (Contradiction, error)
	RecalculateFacts(ctx context.Context, factIDs []string) ([]string, error)
}

type Service struct {
	repo Repo
}

func NewService(repo Repo) *Service {
	return &Service{repo: repo}
}

func (service *Service) RecalculateFacts(ctx context.Context, factIDs []string) ([]string, error) {
	return service.repo.RecalculateFacts(ctx, factIDs)
}

func (service *Service) GetCoverage(ctx context.Context, req *kmapv1.GetCoverageRequest) (*kmapv1.GetCoverageResponse, error) {
	cells, err := service.repo.Coverage(ctx, req.GetDomain())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "coverage: %v", err)
	}
	result := make([]*kmapv1.CoverageCell, 0, len(cells))
	for _, cell := range cells {
		counters, err := structpb.NewStruct(map[string]any{
			"docs": float64(cell.Docs), "experiments": float64(cell.Experiments),
			"facts": float64(cell.Facts), "experts": float64(cell.Experts),
			"ru_docs": float64(cell.RuDocs), "foreign_docs": float64(cell.ForeignDocs),
			"validated_facts": float64(cell.ValidatedFacts), "last_source_year": float64(cell.LastSourceYear),
			"material": cell.MaterialName, "process": cell.ProcessName,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "encode counters: %v", err)
		}
		components, err := structpb.NewStruct(toAnyMap(cell.ScoreComponents))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "encode score components: %v", err)
		}
		result = append(result, &kmapv1.CoverageCell{
			Id: cell.ID, Domain: cell.Domain, MaterialId: cell.MaterialName, ProcessId: cell.ProcessName,
			ConditionKey: cell.ConditionKey, Score: cell.Score, GapFlag: cell.GapFlag,
			GapReasons: cell.GapReasons, Counters: counters, ScoreComponents: components,
		})
	}
	return &kmapv1.GetCoverageResponse{Cells: result}, nil
}

func (service *Service) GetContradictions(ctx context.Context, req *kmapv1.GetContradictionsRequest) (*kmapv1.GetContradictionsResponse, error) {
	items, err := service.repo.Contradictions(ctx, req.GetStatus(), req.GetEntityId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "contradictions: %v", err)
	}
	result := make([]*kmapv1.Contradiction, 0, len(items))
	for _, item := range items {
		message, err := contradictionMessage(item)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "encode contradiction: %v", err)
		}
		result = append(result, message)
	}
	return &kmapv1.GetContradictionsResponse{Contradictions: result, Page: &kmapv1.PageResponse{}}, nil
}

func (service *Service) DecideContradiction(ctx context.Context, req *kmapv1.DecideContradictionRequest) (*kmapv1.DecideContradictionResponse, error) {
	decision := mapDecision(req.GetDecision())
	updated, err := service.repo.DecideContradiction(ctx, req.GetContradictionId(), decision, req.GetComment())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "decide contradiction: %v", err)
	}
	message, err := contradictionMessage(updated)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode contradiction: %v", err)
	}
	return &kmapv1.DecideContradictionResponse{Contradiction: message}, nil
}

func contradictionMessage(item Contradiction) (*kmapv1.Contradiction, error) {
	payload, err := structpb.NewStruct(map[string]any{
		"subject": item.Subject, "parameter": item.Parameter,
		"aStatement": item.AStatement, "bStatement": item.BStatement,
		"cause": item.Cause, "confounders": toAnySlice(item.Confounders),
	})
	if err != nil {
		return nil, err
	}
	return &kmapv1.Contradiction{
		Id: item.ID, ClusterId: item.ClusterID, Status: item.Status,
		Dtype: item.Dtype, Severity: item.Severity, Payload: payload,
	}, nil
}

func mapDecision(decision string) string {
	switch decision {
	case "confirmed":
		return "expert_confirmed"
	case "rejected":
		return "expert_rejected"
	case "resolved":
		return "resolved"
	default:
		return decision
	}
}

func toAnyMap(values map[string]float64) map[string]any {
	result := make(map[string]any, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func toAnySlice(values []string) []any {
	result := make([]any, len(values))
	for index, value := range values {
		result[index] = value
	}
	return result
}
