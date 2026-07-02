package app

import (
	"context"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (service *Service) Search(context.Context, *kmapv1.SearchRequest) (*kmapv1.SearchResponse, error) {
	return &kmapv1.SearchResponse{
		Evidence: &kmapv1.EvidencePack{
			Graph: &kmapv1.Graph{},
		},
	}, nil
}

func (service *Service) EgoGraph(context.Context, *kmapv1.EgoGraphRequest) (*kmapv1.EgoGraphResponse, error) {
	return &kmapv1.EgoGraphResponse{
		Graph: &kmapv1.Graph{},
	}, nil
}

func (service *Service) ListExperts(context.Context, *kmapv1.ListExpertsRequest) (*kmapv1.ListExpertsResponse, error) {
	return &kmapv1.ListExpertsResponse{
		Page: &kmapv1.PageResponse{},
	}, nil
}
