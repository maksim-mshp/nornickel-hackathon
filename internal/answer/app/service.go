package app

import (
	"context"
	"encoding/json"
	"fmt"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

const deltaWordsPerChunk = 8

type SearchClient interface {
	Search(ctx context.Context, in *kmapv1.SearchRequest, opts ...grpc.CallOption) (*kmapv1.SearchResponse, error)
}

type Emit func(*kmapv1.AskResponse) error

type Service struct {
	search SearchClient
}

func NewService(search SearchClient) *Service {
	return &Service{search: search}
}

func (service *Service) Ask(ctx context.Context, req *kmapv1.AskRequest, emit Emit) error {
	question := req.GetQuestion()
	if question == "" {
		return status.Error(codes.InvalidArgument, "question is required")
	}

	plan, err := buildPlan(question)
	if err != nil {
		return status.Errorf(codes.Internal, "build plan: %v", err)
	}
	if err := emit(&kmapv1.AskResponse{Type: "plan", Plan: plan}); err != nil {
		return err
	}

	searchResp, err := service.search.Search(ctx, &kmapv1.SearchRequest{Plan: plan, Principal: req.GetPrincipal()})
	if err != nil {
		return status.Errorf(codes.Internal, "search: %v", err)
	}
	pack := searchResp.GetEvidence()
	if err := emit(&kmapv1.AskResponse{Type: "evidence", Evidence: pack}); err != nil {
		return err
	}

	summary, methods, confidence := synthesize(pack)
	for _, delta := range chunkDeltas(summary, deltaWordsPerChunk) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := emit(&kmapv1.AskResponse{Type: "answer.delta", Delta: delta}); err != nil {
			return err
		}
	}

	guard := runGuard(summary, pack)
	payload, err := methodsStruct(methods)
	if err != nil {
		return status.Errorf(codes.Internal, "encode answer: %v", err)
	}
	answer := &kmapv1.AnswerDoc{
		Summary:    summary,
		Confidence: confidence,
		Payload:    payload,
		Guard: &kmapv1.GuardReport{
			NumbersChecked: uint32(guard.numbersChecked),
			Violations:     uint32(guard.violations),
			Degraded:       guard.violations > 0,
		},
	}
	return emit(&kmapv1.AskResponse{Type: "answer.done", Answer: answer})
}

func methodsStruct(methods []methodView) (*structpb.Struct, error) {
	data, err := json.Marshal(map[string]any{"methods": methods})
	if err != nil {
		return nil, fmt.Errorf("marshal methods: %w", err)
	}
	var asMap map[string]any
	if err := json.Unmarshal(data, &asMap); err != nil {
		return nil, fmt.Errorf("unmarshal methods: %w", err)
	}
	return structpb.NewStruct(asMap)
}
