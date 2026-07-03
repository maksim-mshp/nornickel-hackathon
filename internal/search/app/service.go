package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type Repo interface {
	ExpandEntityIDs(ctx context.Context, slugs []string) ([]string, error)
	Facts(ctx context.Context, entityIDs []string) ([]Fact, error)
	Consensus(ctx context.Context, entityIDs []string) ([]Consensus, error)
	Contradictions(ctx context.Context, entityIDs []string) ([]Contradiction, error)
	Gaps(ctx context.Context, entityIDs []string) ([]GapCell, error)
	Experts(ctx context.Context, entityIDs []string) ([]Expert, error)
	EgoGraph(ctx context.Context, entityIDs []string) ([]GraphNode, []GraphEdge, error)
}

type Service struct {
	repo        Repo
	ranking     Ranking
	currentYear int
}

func NewService(repo Repo, ranking Ranking, currentYear int) *Service {
	return &Service{repo: repo, ranking: ranking, currentYear: currentYear}
}

func (service *Service) Search(ctx context.Context, req *kmapv1.SearchRequest) (*kmapv1.SearchResponse, error) {
	slugs := planSlugs(req.GetPlan())
	pack, err := service.buildPack(ctx, slugs)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "build evidence pack: %v", err)
	}
	proto, err := pack.toProto()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode evidence pack: %v", err)
	}
	return &kmapv1.SearchResponse{Evidence: proto}, nil
}

func (service *Service) EgoGraph(ctx context.Context, req *kmapv1.EgoGraphRequest) (*kmapv1.EgoGraphResponse, error) {
	ids, err := service.repo.ExpandEntityIDs(ctx, []string{req.GetEntityId()})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "resolve entity: %v", err)
	}
	nodes, edges, err := service.repo.EgoGraph(ctx, ids)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "ego graph: %v", err)
	}
	return &kmapv1.EgoGraphResponse{Graph: buildGraph(nodes, edges)}, nil
}

func (service *Service) ListExperts(ctx context.Context, req *kmapv1.ListExpertsRequest) (*kmapv1.ListExpertsResponse, error) {
	anchor := req.GetEntityId()
	if anchor == "" {
		anchor = req.GetTopic()
	}
	ids, err := service.repo.ExpandEntityIDs(ctx, []string{anchor})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "resolve entity: %v", err)
	}
	experts, err := service.repo.Experts(ctx, ids)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list experts: %v", err)
	}
	messages, err := toExpertMessages(experts)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode experts: %v", err)
	}
	return &kmapv1.ListExpertsResponse{Experts: messages, Page: &kmapv1.PageResponse{}}, nil
}

func (service *Service) buildPack(ctx context.Context, slugs []string) (EvidencePack, error) {
	if len(slugs) == 0 {
		return EvidencePack{}, nil
	}
	entityIDs, err := service.repo.ExpandEntityIDs(ctx, slugs)
	if err != nil {
		return EvidencePack{}, err
	}
	if len(entityIDs) == 0 {
		return EvidencePack{}, nil
	}

	facts, err := service.repo.Facts(ctx, entityIDs)
	if err != nil {
		return EvidencePack{}, err
	}
	service.rankFacts(facts, slugs)
	refByID := assignRefs(facts)

	consensus, err := service.repo.Consensus(ctx, entityIDs)
	if err != nil {
		return EvidencePack{}, err
	}
	contradictions, err := service.repo.Contradictions(ctx, entityIDs)
	if err != nil {
		return EvidencePack{}, err
	}
	contradictions = remapContradictions(contradictions, refByID)

	gaps, err := service.repo.Gaps(ctx, entityIDs)
	if err != nil {
		return EvidencePack{}, err
	}
	experts, err := service.repo.Experts(ctx, entityIDs)
	if err != nil {
		return EvidencePack{}, err
	}
	nodes, edges, err := service.repo.EgoGraph(ctx, entityIDs)
	if err != nil {
		return EvidencePack{}, err
	}

	return EvidencePack{
		Facts:          facts,
		Consensus:      consensus,
		Contradictions: contradictions,
		Gaps:           gaps,
		Experts:        experts,
		GraphNodes:     nodes,
		GraphEdges:     edges,
		Stats:          computeStats(facts),
	}, nil
}

func (service *Service) rankFacts(facts []Fact, slugs []string) {
	planSet := toSet(slugs)
	for index := range facts {
		fact := &facts[index]
		match := 0.6
		if planSet[fact.Subject.Slug] || planSet[fact.Parameter.Slug] {
			match = 1.0
		}
		fact.ScoreComponents = service.ranking.score(match, fact.Confidence, fact.Provenance.DocType, fact.ValidationStatus, fact.Provenance.Year, service.currentYear)
		fact.Score = service.ranking.finalScore(fact.ScoreComponents)
	}
	sort.SliceStable(facts, func(i, j int) bool {
		return facts[i].Score > facts[j].Score
	})
}

func assignRefs(facts []Fact) map[string]string {
	refByID := make(map[string]string, len(facts))
	for index := range facts {
		ref := fmt.Sprintf("F%d", index+1)
		facts[index].Ref = ref
		refByID[facts[index].ID] = ref
	}
	return refByID
}

func remapContradictions(contradictions []Contradiction, refByID map[string]string) []Contradiction {
	result := make([]Contradiction, 0, len(contradictions))
	for _, contradiction := range contradictions {
		aRef, aOK := refByID[contradiction.AFactRef]
		bRef, bOK := refByID[contradiction.BFactRef]
		if !aOK || !bOK {
			continue
		}
		contradiction.AFactRef = aRef
		contradiction.BFactRef = bRef
		result = append(result, contradiction)
	}
	return result
}

func computeStats(facts []Fact) EvidenceStats {
	type docInfo struct {
		geography string
		year      int
	}
	docs := map[string]docInfo{}
	for _, fact := range facts {
		docs[fact.Provenance.DocumentID] = docInfo{geography: fact.Geography, year: fact.Provenance.Year}
	}
	stats := EvidenceStats{Sources: len(docs)}
	for _, info := range docs {
		switch info.geography {
		case "ru":
			stats.RuSources++
		case "foreign":
			stats.ForeignSources++
		}
		if info.year == 0 {
			continue
		}
		if stats.YearFrom == 0 || info.year < stats.YearFrom {
			stats.YearFrom = info.year
		}
		if info.year > stats.YearTo {
			stats.YearTo = info.year
		}
	}
	return stats
}

func planSlugs(plan *kmapv1.QueryPlan) []string {
	if plan == nil || plan.GetEntities() == nil {
		return nil
	}
	var slugs []string
	fields := plan.GetEntities().GetFields()
	for _, group := range []string{"materials", "processes", "properties"} {
		list := fields[group].GetListValue()
		if list == nil {
			continue
		}
		for _, item := range list.GetValues() {
			if slug := item.GetStructValue().GetFields()["slug"].GetStringValue(); slug != "" {
				slugs = append(slugs, slug)
			}
		}
	}
	return slugs
}

func toSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

func (pack EvidencePack) toProto() (*kmapv1.EvidencePack, error) {
	facts := make([]*kmapv1.Fact, 0, len(pack.Facts))
	for _, item := range pack.Facts {
		payload, err := toStruct(item)
		if err != nil {
			return nil, err
		}
		facts = append(facts, &kmapv1.Fact{Id: item.ID, Kind: "numeric", Payload: payload})
	}
	consensus, err := toStructList(pack.Consensus)
	if err != nil {
		return nil, err
	}
	contradictions, err := toStructList(pack.Contradictions)
	if err != nil {
		return nil, err
	}
	gaps, err := toStructList(pack.Gaps)
	if err != nil {
		return nil, err
	}
	experts, err := toExpertMessages(pack.Experts)
	if err != nil {
		return nil, err
	}
	stats, err := toStruct(pack.Stats)
	if err != nil {
		return nil, err
	}
	return &kmapv1.EvidencePack{
		Facts:          facts,
		Consensus:      consensus,
		Contradictions: contradictions,
		Gaps:           gaps,
		Experts:        experts,
		Graph:          buildGraph(pack.GraphNodes, pack.GraphEdges),
		Stats:          stats,
	}, nil
}

func buildGraph(nodes []GraphNode, edges []GraphEdge) *kmapv1.Graph {
	graphNodes := make([]*kmapv1.GraphNode, 0, len(nodes))
	for _, node := range nodes {
		graphNodes = append(graphNodes, &kmapv1.GraphNode{Id: node.ID, Type: node.Type, Label: node.Label})
	}
	graphEdges := make([]*kmapv1.GraphEdge, 0, len(edges))
	for _, edge := range edges {
		graphEdges = append(graphEdges, &kmapv1.GraphEdge{
			Id: edge.ID, Src: edge.Src, Dst: edge.Dst, Rel: edge.Rel,
			Weight: edge.Weight, Confidence: edge.Confidence, Contradiction: edge.Contradiction,
		})
	}
	return &kmapv1.Graph{Nodes: graphNodes, Edges: graphEdges}
}

func toExpertMessages(experts []Expert) ([]*kmapv1.Expert, error) {
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
