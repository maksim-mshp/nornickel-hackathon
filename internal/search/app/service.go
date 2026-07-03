package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (service *Service) Search(_ context.Context, req *kmapv1.SearchRequest) (*kmapv1.SearchResponse, error) {
	scn := selectScenario(req.GetPlan())
	pack, err := scn.evidencePack()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "build evidence pack: %v", err)
	}
	return &kmapv1.SearchResponse{Evidence: pack}, nil
}

func (service *Service) EgoGraph(_ context.Context, _ *kmapv1.EgoGraphRequest) (*kmapv1.EgoGraphResponse, error) {
	scn := catholyteScenario()
	return &kmapv1.EgoGraphResponse{Graph: scn.egoGraph()}, nil
}

func (service *Service) ListExperts(_ context.Context, _ *kmapv1.ListExpertsRequest) (*kmapv1.ListExpertsResponse, error) {
	scn := catholyteScenario()
	experts, err := toExpertMessages(scn.experts)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode experts: %v", err)
	}
	return &kmapv1.ListExpertsResponse{Experts: experts, Page: &kmapv1.PageResponse{}}, nil
}

func (scn scenario) evidencePack() (*kmapv1.EvidencePack, error) {
	facts := make([]*kmapv1.Fact, 0, len(scn.facts))
	for _, item := range scn.facts {
		payload, err := toStruct(item)
		if err != nil {
			return nil, err
		}
		facts = append(facts, &kmapv1.Fact{Id: item.ID, Kind: "numeric", Payload: payload})
	}

	consensus, err := toStructList(scn.consensus)
	if err != nil {
		return nil, err
	}
	contradictions, err := toStructList(scn.contradictions)
	if err != nil {
		return nil, err
	}
	gaps, err := toStructList(scn.gaps)
	if err != nil {
		return nil, err
	}
	experts, err := toExpertMessages(scn.experts)
	if err != nil {
		return nil, err
	}
	stats, err := toStruct(scn.stats)
	if err != nil {
		return nil, err
	}

	return &kmapv1.EvidencePack{
		Facts:          facts,
		Consensus:      consensus,
		Contradictions: contradictions,
		Gaps:           gaps,
		Experts:        experts,
		Graph:          scn.egoGraph(),
		Stats:          stats,
	}, nil
}

func (scn scenario) egoGraph() *kmapv1.Graph {
	nodes := []*kmapv1.GraphNode{}
	edges := []*kmapv1.GraphEdge{}
	seen := map[string]bool{}

	addNode := func(ref entityRef, kind string) {
		if ref.Slug == "" || seen[ref.Slug] {
			return
		}
		seen[ref.Slug] = true
		nodes = append(nodes, &kmapv1.GraphNode{Id: ref.Slug, Type: kind, Label: ref.Name})
	}

	for _, process := range scn.processes {
		addNode(process, "process")
	}
	for _, material := range scn.materials {
		addNode(material, "material")
	}
	for _, property := range scn.properties {
		addNode(property, "property")
	}
	for _, item := range scn.facts {
		addNode(item.Subject, subjectKind(item.Subject.Slug))
		addNode(item.Parameter, "parameter")
		if len(scn.processes) > 0 {
			edges = append(edges, &kmapv1.GraphEdge{
				Id:     "edge:" + item.Subject.Slug + ":" + item.Parameter.Slug,
				Src:    item.Subject.Slug,
				Dst:    item.Parameter.Slug,
				Rel:    "OPERATES_AT",
				Weight: 1,
			})
		}
	}
	for _, material := range scn.materials {
		for _, process := range scn.processes {
			edges = append(edges, &kmapv1.GraphEdge{
				Id:     "edge:" + process.Slug + ":" + material.Slug,
				Src:    process.Slug,
				Dst:    material.Slug,
				Rel:    "USES_MATERIAL",
				Weight: 1,
			})
		}
	}

	return &kmapv1.Graph{Nodes: nodes, Edges: edges}
}

func subjectKind(slug string) string {
	switch {
	case strings.HasPrefix(slug, "experiment:"):
		return "experiment"
	case strings.HasPrefix(slug, "technology:"):
		return "technology"
	default:
		return "process"
	}
}

func toExpertMessages(experts []expert) ([]*kmapv1.Expert, error) {
	result := make([]*kmapv1.Expert, 0, len(experts))
	for _, item := range experts {
		evidence, err := toStruct(item)
		if err != nil {
			return nil, err
		}
		result = append(result, &kmapv1.Expert{
			PersonId: item.ID,
			Name:     item.Name,
			Weight:   item.Weight,
			Evidence: evidence,
		})
	}
	return result, nil
}

func toStruct(value any) (*structpb.Struct, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal value: %w", err)
	}
	var asMap map[string]any
	if err := json.Unmarshal(data, &asMap); err != nil {
		return nil, fmt.Errorf("unmarshal value: %w", err)
	}
	result, err := structpb.NewStruct(asMap)
	if err != nil {
		return nil, fmt.Errorf("build struct: %w", err)
	}
	return result, nil
}

func toStructList[T any](values []T) ([]*structpb.Struct, error) {
	result := make([]*structpb.Struct, 0, len(values))
	for _, value := range values {
		item, err := toStruct(value)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}
