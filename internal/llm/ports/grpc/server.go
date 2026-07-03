package grpc

import (
	"context"
	"encoding/json"
	"errors"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"github.com/maksim-mshp/nornickel-hackathon/internal/llm/app"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type Server struct {
	kmapv1.UnimplementedLLMServiceServer
	service *app.Service
}

func NewServer(service *app.Service) *Server {
	return &Server{service: service}
}

func (server *Server) RegisterGRPC(registrar grpc.ServiceRegistrar) {
	kmapv1.RegisterLLMServiceServer(registrar, server)
}

func (server *Server) Complete(ctx context.Context, req *kmapv1.CompleteRequest) (*kmapv1.CompleteResponse, error) {
	result, err := server.service.Complete(ctx, req.GetTask(), payloadMap(req.GetPayload()))
	if err != nil {
		return nil, mapError(err)
	}
	return &kmapv1.CompleteResponse{
		Json:         toStruct(result),
		Model:        result.Model,
		InputTokens:  uint32(result.InputTokens),
		OutputTokens: uint32(result.OutputTokens),
		Valid:        result.Valid,
	}, nil
}

func (server *Server) CompleteStream(req *kmapv1.CompleteStreamRequest, stream kmapv1.LLMService_CompleteStreamServer) error {
	result, err := server.service.Complete(stream.Context(), req.GetTask(), payloadMap(req.GetPayload()))
	if err != nil {
		return mapError(err)
	}
	return stream.Send(&kmapv1.CompleteStreamResponse{
		Delta: result.Content,
		Done:  true,
		Json:  toStruct(result),
	})
}

func payloadMap(value *structpb.Struct) map[string]any {
	if value == nil {
		return nil
	}
	return value.AsMap()
}

func toStruct(result *app.Result) *structpb.Struct {
	if result.IsJSON && result.Valid {
		var raw map[string]any
		if err := json.Unmarshal([]byte(result.Content), &raw); err == nil {
			if st, err := structpb.NewStruct(raw); err == nil {
				return st
			}
		}
	}
	st, _ := structpb.NewStruct(map[string]any{"text": result.Content})
	return st
}

func mapError(err error) error {
	switch {
	case errors.Is(err, app.ErrUnknownTask):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, app.ErrModelNotAllowed), errors.Is(err, app.ErrProviderNotConfigured):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, app.ErrEmptyResponse):
		return status.Error(codes.Internal, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
