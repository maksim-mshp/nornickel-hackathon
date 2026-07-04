package grpc

import (
	"context"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"github.com/maksim-mshp/nornickel-hackathon/internal/answer/app"
	"google.golang.org/grpc"
)

type Server struct {
	kmapv1.UnimplementedAnswerServiceServer
	service *app.Service
}

func NewServer(search kmapv1.SearchServiceClient, options ...app.Option) *Server {
	return &Server{service: app.NewService(search, options...)}
}

func (server *Server) RegisterGRPC(registrar grpc.ServiceRegistrar) {
	kmapv1.RegisterAnswerServiceServer(registrar, server)
}

func (server *Server) Ask(req *kmapv1.AskRequest, stream kmapv1.AnswerService_AskServer) error {
	return server.service.Ask(stream.Context(), req, stream.Send)
}

func (server *Server) ParseQuery(_ context.Context, req *kmapv1.ParseQueryRequest) (*kmapv1.ParseQueryResponse, error) {
	plan, err := server.service.ParseQuery(req.GetQuestion())
	if err != nil {
		return nil, err
	}
	return &kmapv1.ParseQueryResponse{Plan: plan}, nil
}
