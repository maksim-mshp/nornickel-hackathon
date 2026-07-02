package grpc

import (
	"context"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"github.com/maksim-mshp/nornickel-hackathon/internal/search/app"
	"google.golang.org/grpc"
)

type Server struct {
	kmapv1.UnimplementedSearchServiceServer
	service *app.Service
}

func NewServer() *Server {
	return &Server{service: app.NewService()}
}

func (server *Server) RegisterGRPC(registrar grpc.ServiceRegistrar) {
	kmapv1.RegisterSearchServiceServer(registrar, server)
}

func (server *Server) Search(ctx context.Context, req *kmapv1.SearchRequest) (*kmapv1.SearchResponse, error) {
	return server.service.Search(ctx, req)
}

func (server *Server) EgoGraph(ctx context.Context, req *kmapv1.EgoGraphRequest) (*kmapv1.EgoGraphResponse, error) {
	return server.service.EgoGraph(ctx, req)
}

func (server *Server) ListExperts(ctx context.Context, req *kmapv1.ListExpertsRequest) (*kmapv1.ListExpertsResponse, error) {
	return server.service.ListExperts(ctx, req)
}
