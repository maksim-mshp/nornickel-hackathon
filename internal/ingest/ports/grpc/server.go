package grpc

import (
	"context"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"github.com/maksim-mshp/nornickel-hackathon/internal/ingest/app"
	"google.golang.org/grpc"
)

type Server struct {
	kmapv1.UnimplementedIngestServiceServer
	service *app.Service
}

func NewServer() *Server {
	return &Server{service: app.NewService()}
}

func (server *Server) RegisterGRPC(registrar grpc.ServiceRegistrar) {
	kmapv1.RegisterIngestServiceServer(registrar, server)
}

func (server *Server) RegisterDocument(ctx context.Context, req *kmapv1.RegisterDocumentRequest) (*kmapv1.RegisterDocumentResponse, error) {
	return server.service.RegisterDocument(ctx, req)
}

func (server *Server) GetStatus(ctx context.Context, req *kmapv1.GetStatusRequest) (*kmapv1.GetStatusResponse, error) {
	return server.service.GetStatus(ctx, req)
}
