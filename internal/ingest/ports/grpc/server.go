package grpc

import (
	"context"
	"errors"
	"strconv"

	"github.com/google/uuid"
	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"github.com/maksim-mshp/nornickel-hackathon/internal/ingest/app"
	"github.com/maksim-mshp/nornickel-hackathon/internal/ingest/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type Server struct {
	kmapv1.UnimplementedIngestServiceServer
	service *app.Service
}

func NewServer(service *app.Service) *Server {
	return &Server{service: service}
}

func (server *Server) RegisterGRPC(registrar grpc.ServiceRegistrar) {
	kmapv1.RegisterIngestServiceServer(registrar, server)
}

func (server *Server) RegisterDocument(ctx context.Context, req *kmapv1.RegisterDocumentRequest) (*kmapv1.RegisterDocumentResponse, error) {
	meta := readDeclaredMeta(req.GetDeclaredMeta())

	result, err := server.service.RegisterDocument(ctx, app.RegisterCommand{
		Title:       req.GetTitle(),
		BlobURI:     req.GetBlobUri(),
		SHA256:      req.GetSha256(),
		DocType:     strValue(meta, "doc_type"),
		Lang:        strValue(meta, "lang"),
		Geography:   strValue(meta, "geography"),
		AccessLevel: strValue(meta, "access_level"),
		Year:        intValue(meta, "year"),
		UploadedBy:  req.GetPrincipal().GetUserId(),
		Meta:        meta,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &kmapv1.RegisterDocumentResponse{
		DocumentId: result.DocumentID.String(),
		Version:    int32(result.Version),
		Status:     result.Status,
		Duplicate:  result.Duplicate,
	}, nil
}

func (server *Server) GetStatus(ctx context.Context, req *kmapv1.GetStatusRequest) (*kmapv1.GetStatusResponse, error) {
	documentID, err := uuid.Parse(req.GetDocument().GetDocumentId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid document_id")
	}

	result, err := server.service.GetStatus(ctx, documentID)
	if err != nil {
		return nil, mapError(err)
	}

	return &kmapv1.GetStatusResponse{
		Document: &kmapv1.DocumentRef{
			DocumentId: result.Document.ID.String(),
			Version:    int32(result.Document.Version),
		},
		Status: result.Document.Status,
		Stages: toProtoStages(result.Stages),
	}, nil
}

func toProtoStages(stages []domain.Stage) []*kmapv1.IngestStage {
	out := make([]*kmapv1.IngestStage, 0, len(stages))
	for _, stage := range stages {
		out = append(out, &kmapv1.IngestStage{
			Stage:   stage.Stage,
			Status:  stage.Status,
			Attempt: uint32(stage.Attempt),
			Error:   stage.Error,
		})
	}
	return out
}

func readDeclaredMeta(value *structpb.Struct) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value.AsMap()
}

func strValue(meta map[string]any, key string) string {
	value, ok := meta[key].(string)
	if !ok {
		return ""
	}
	return value
}

func intValue(meta map[string]any, key string) int {
	switch value := meta[key].(type) {
	case float64:
		return int(value)
	case string:
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return 0
		}
		return parsed
	default:
		return 0
	}
}

func mapError(err error) error {
	switch {
	case errors.Is(err, domain.ErrSHA256Required),
		errors.Is(err, domain.ErrBlobURIRequired),
		errors.Is(err, domain.ErrInvalidDocType),
		errors.Is(err, domain.ErrInvalidGeography),
		errors.Is(err, domain.ErrInvalidAccessLevel):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrDocumentNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
