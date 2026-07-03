package app

import (
	"context"
	"os"
	"testing"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"github.com/maksim-mshp/nornickel-hackathon/internal/llm/adapters/openai"
	llmapp "github.com/maksim-mshp/nornickel-hackathon/internal/llm/app"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/config"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

type inProcessLLM struct {
	service *llmapp.Service
}

func (adapter inProcessLLM) Complete(ctx context.Context, in *kmapv1.CompleteRequest, _ ...grpc.CallOption) (*kmapv1.CompleteResponse, error) {
	result, err := adapter.service.Complete(ctx, in.GetTask(), in.GetPayload().AsMap())
	if err != nil {
		return nil, err
	}
	json, _ := structpb.NewStruct(map[string]any{"text": result.Content})
	return &kmapv1.CompleteResponse{Json: json, Model: result.Model}, nil
}

func TestLiveLLMSynthesisPassesGuard(t *testing.T) {
	key := os.Getenv("KMAP_YANDEX_KEY")
	folder := os.Getenv("KMAP_YANDEX_FOLDER")
	if key == "" || folder == "" {
		t.Skip("set KMAP_YANDEX_KEY and KMAP_YANDEX_FOLDER to run the live end-to-end LLM synthesis test")
	}

	cfg := config.LLM{
		DefaultProvider: "yandex",
		Providers: map[string]config.LLMProvider{
			"yandex": {BaseURL: "https://ai.api.cloud.yandex.net/v1", AuthScheme: "api-key", FolderID: folder},
		},
		Tasks: map[string]config.LLMTask{
			"synthesize_answer": {Model: "deepseek-v4-flash/latest", MaxTokens: 2000, Temperature: 0.3, TimeoutS: 60},
		},
	}
	llmClient := openai.New(cfg.Providers["yandex"].BaseURL, key, "api-key", folder)
	llmService, err := llmapp.New(cfg, llmClient)
	if err != nil {
		t.Fatalf("build llm service: %v", err)
	}

	pack := samplecatholytePack(t)
	synth := NewLLMSynthesizer(inProcessLLM{service: llmService})
	result, err := synth.Synthesize(context.Background(), "оптимальная скорость циркуляции католита?", pack)
	if err != nil {
		t.Fatalf("live synthesize: %v", err)
	}
	if result.Summary == "" {
		t.Fatal("empty synthesis")
	}

	guard := runGuard(result.Summary, pack)
	t.Logf("summary=%q numbers_checked=%d violations=%d", result.Summary, guard.numbersChecked, guard.violations)
}
