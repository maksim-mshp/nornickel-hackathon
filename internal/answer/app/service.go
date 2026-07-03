package app

import (
	"context"
	"crypto/sha256"
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

type CachedAnswer struct {
	Plan     *kmapv1.QueryPlan
	Evidence *kmapv1.EvidencePack
	Answer   *kmapv1.AnswerDoc
}

type Cache interface {
	Get(ctx context.Context, key []byte) (*CachedAnswer, bool, error)
	Put(ctx context.Context, key []byte, value *CachedAnswer) error
}

type Emit func(*kmapv1.AskResponse) error

type Synthesis struct {
	Summary    string
	Methods    []methodView
	Confidence float64
}

type Synthesizer interface {
	Synthesize(ctx context.Context, pack *kmapv1.EvidencePack) (Synthesis, error)
}

type extractiveSynthesizer struct{}

func (extractiveSynthesizer) Synthesize(_ context.Context, pack *kmapv1.EvidencePack) (Synthesis, error) {
	return extractiveSynthesis(pack), nil
}

func extractiveSynthesis(pack *kmapv1.EvidencePack) Synthesis {
	summary, methods, confidence := synthesize(pack)
	return Synthesis{Summary: summary, Methods: methods, Confidence: confidence}
}

type Service struct {
	search SearchClient
	cache  Cache
	synth  Synthesizer
}

type Option func(*Service)

func WithCache(cache Cache) Option {
	return func(service *Service) {
		service.cache = cache
	}
}

func WithSynthesizer(synth Synthesizer) Option {
	return func(service *Service) {
		service.synth = synth
	}
}

func NewService(search SearchClient, options ...Option) *Service {
	service := &Service{search: search, synth: extractiveSynthesizer{}}
	for _, option := range options {
		option(service)
	}
	return service
}

func (service *Service) Ask(ctx context.Context, req *kmapv1.AskRequest, emit Emit) error {
	question := req.GetQuestion()
	if question == "" {
		return status.Error(codes.InvalidArgument, "question is required")
	}

	key := cacheKey(question, req.GetPrincipal().GetDocAccess())
	if service.cache != nil {
		if cached, ok, err := service.cache.Get(ctx, key); err == nil && ok {
			return replayCached(ctx, cached, emit)
		}
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

	result, err := service.synth.Synthesize(ctx, pack)
	if err != nil {
		return status.Errorf(codes.Internal, "synthesize: %v", err)
	}
	guard := runGuard(result.Summary, pack)
	degraded := false
	if guard.violations > 0 {
		result = extractiveSynthesis(pack)
		guard = runGuard(result.Summary, pack)
		degraded = true
	}

	for _, delta := range chunkDeltas(result.Summary, deltaWordsPerChunk) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := emit(&kmapv1.AskResponse{Type: "answer.delta", Delta: delta}); err != nil {
			return err
		}
	}

	payload, err := methodsStruct(result.Methods)
	if err != nil {
		return status.Errorf(codes.Internal, "encode answer: %v", err)
	}
	answer := &kmapv1.AnswerDoc{
		Summary:    result.Summary,
		Confidence: result.Confidence,
		Payload:    payload,
		Guard: &kmapv1.GuardReport{
			NumbersChecked: uint32(guard.numbersChecked),
			Violations:     uint32(guard.violations),
			Degraded:       degraded,
		},
	}

	if service.cache != nil && guard.violations == 0 && !degraded {
		_ = service.cache.Put(ctx, key, &CachedAnswer{Plan: plan, Evidence: pack, Answer: answer})
	}

	return emit(&kmapv1.AskResponse{Type: "answer.done", Answer: answer})
}

func replayCached(ctx context.Context, cached *CachedAnswer, emit Emit) error {
	if err := emit(&kmapv1.AskResponse{Type: "plan", Plan: cached.Plan}); err != nil {
		return err
	}
	if err := emit(&kmapv1.AskResponse{Type: "evidence", Evidence: cached.Evidence}); err != nil {
		return err
	}
	for _, delta := range chunkDeltas(cached.Answer.GetSummary(), deltaWordsPerChunk) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := emit(&kmapv1.AskResponse{Type: "answer.delta", Delta: delta}); err != nil {
			return err
		}
	}
	return emit(&kmapv1.AskResponse{Type: "answer.done", Answer: cached.Answer})
}

func cacheKey(question string, docAccess string) []byte {
	digest := sha256.Sum256([]byte(question + "\x00" + docAccess))
	return digest[:]
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
