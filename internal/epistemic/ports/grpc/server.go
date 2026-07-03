package grpc

import (
	"context"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"github.com/maksim-mshp/nornickel-hackathon/internal/epistemic/app"
	"google.golang.org/grpc"
)

type Server struct {
	kmapv1.UnimplementedEpistemicServiceServer
	service *app.Service
}

func NewServer(service *app.Service) *Server {
	return &Server{service: service}
}

func (server *Server) RegisterGRPC(registrar grpc.ServiceRegistrar) {
	kmapv1.RegisterEpistemicServiceServer(registrar, server)
}

func (server *Server) GetCoverage(ctx context.Context, req *kmapv1.GetCoverageRequest) (*kmapv1.GetCoverageResponse, error) {
	return server.service.GetCoverage(ctx, req)
}

func (server *Server) GetContradictions(ctx context.Context, req *kmapv1.GetContradictionsRequest) (*kmapv1.GetContradictionsResponse, error) {
	return server.service.GetContradictions(ctx, req)
}

func (server *Server) DecideContradiction(ctx context.Context, req *kmapv1.DecideContradictionRequest) (*kmapv1.DecideContradictionResponse, error) {
	return server.service.DecideContradiction(ctx, req)
}
