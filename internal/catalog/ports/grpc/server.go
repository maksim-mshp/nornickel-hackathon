package grpc

import (
	"context"

	"github.com/google/uuid"
	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"github.com/maksim-mshp/nornickel-hackathon/internal/catalog/app"
	"google.golang.org/grpc"
)

type Server struct {
	kmapv1.UnimplementedCatalogServiceServer
	service *app.Service
}

func NewServer(service *app.Service) *Server {
	return &Server{service: service}
}

func (server *Server) RegisterGRPC(registrar grpc.ServiceRegistrar) {
	kmapv1.RegisterCatalogServiceServer(registrar, server)
}

func (server *Server) CommitExtraction(ctx context.Context, req *kmapv1.CommitExtractionRequest) (*kmapv1.CommitExtractionResponse, error) {
	result, err := server.service.CommitExtraction(ctx, req.GetBundleUri())
	if err != nil {
		return nil, err
	}
	return &kmapv1.CommitExtractionResponse{
		DocumentId:  result.DocumentID.String(),
		FactIds:     uuidStrings(result.FactIDs),
		EntityIds:   uuidStrings(result.EntityIDs),
		ClusterKeys: result.ClusterKeys,
	}, nil
}

func (server *Server) ResolveEntities(ctx context.Context, req *kmapv1.ResolveEntitiesRequest) (*kmapv1.ResolveEntitiesResponse, error) {
	resolutions, err := server.service.ResolveEntities(ctx, req.GetNames())
	if err != nil {
		return nil, err
	}
	results := make([]*kmapv1.EntityResolution, 0, len(resolutions))
	for _, resolution := range resolutions {
		results = append(results, &kmapv1.EntityResolution{
			Input:         resolution.Input,
			EntityId:      resolution.EntityID,
			Slug:          resolution.Slug,
			CanonicalName: resolution.Name,
			Confidence:    resolution.Confidence,
			Status:        resolution.Status,
		})
	}
	return &kmapv1.ResolveEntitiesResponse{Results: results}, nil
}

func (server *Server) MergeEntities(ctx context.Context, req *kmapv1.MergeEntitiesRequest) (*kmapv1.MergeEntitiesResponse, error) {
	if err := server.service.MergeEntities(ctx, req.GetEntityId(), req.GetIntoId(), req.GetPrincipal().GetUserId(), req.GetComment()); err != nil {
		return nil, err
	}
	return &kmapv1.MergeEntitiesResponse{EntityId: req.GetEntityId(), IntoId: req.GetIntoId()}, nil
}

func (server *Server) UpdateFactStatus(ctx context.Context, req *kmapv1.UpdateFactStatusRequest) (*kmapv1.UpdateFactStatusResponse, error) {
	if err := server.service.UpdateFactStatus(ctx, req.GetFactId(), req.GetFactKind(), req.GetStatus(), req.GetPrincipal().GetUserId(), req.GetComment()); err != nil {
		return nil, err
	}
	return &kmapv1.UpdateFactStatusResponse{Fact: &kmapv1.Fact{Id: req.GetFactId(), Kind: req.GetFactKind()}}, nil
}

func uuidStrings(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}
