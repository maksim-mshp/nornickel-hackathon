package app

import (
	"context"
	"sync"

	"github.com/google/uuid"
	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	mu        sync.RWMutex
	documents map[string]*kmapv1.GetStatusResponse
}

func NewService() *Service {
	return &Service{documents: map[string]*kmapv1.GetStatusResponse{}}
}

func (service *Service) RegisterDocument(_ context.Context, req *kmapv1.RegisterDocumentRequest) (*kmapv1.RegisterDocumentResponse, error) {
	if len(req.GetSha256()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "sha256 is required")
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate document id: %v", err)
	}

	ref := &kmapv1.DocumentRef{DocumentId: id.String(), Version: 1}
	statusResp := &kmapv1.GetStatusResponse{
		Document: ref,
		Status:   "registered",
		Stages: []*kmapv1.IngestStage{
			{Stage: "register", Status: "done"},
			{Stage: "parse", Status: "pending"},
			{Stage: "extract", Status: "pending"},
			{Stage: "commit", Status: "pending"},
			{Stage: "epistemic", Status: "pending"},
		},
	}

	service.mu.Lock()
	service.documents[ref.GetDocumentId()] = statusResp
	service.mu.Unlock()

	return &kmapv1.RegisterDocumentResponse{
		DocumentId: ref.GetDocumentId(),
		Version:    ref.GetVersion(),
		Status:     statusResp.GetStatus(),
	}, nil
}

func (service *Service) GetStatus(_ context.Context, req *kmapv1.GetStatusRequest) (*kmapv1.GetStatusResponse, error) {
	documentID := req.GetDocument().GetDocumentId()
	if documentID == "" {
		return nil, status.Error(codes.InvalidArgument, "document_id is required")
	}

	service.mu.RLock()
	resp, ok := service.documents[documentID]
	service.mu.RUnlock()
	if !ok {
		return nil, status.Error(codes.NotFound, "document not found")
	}

	return resp, nil
}
