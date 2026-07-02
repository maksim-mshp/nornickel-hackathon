package grpc

import (
	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"google.golang.org/grpc"
)

type Server struct {
	kmapv1.UnimplementedAnswerServiceServer
}

func NewServer() *Server {
	return &Server{}
}

func (server *Server) RegisterGRPC(registrar grpc.ServiceRegistrar) {
	kmapv1.RegisterAnswerServiceServer(registrar, server)
}
