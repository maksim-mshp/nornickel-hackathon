package app

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/config"
)

type fakeChatClient struct {
	result   *ChatResult
	err      error
	gotModel string
}

func (client *fakeChatClient) Complete(_ context.Context, model string, _ []Message, _ Options) (*ChatResult, error) {
	client.gotModel = model
	if client.err != nil {
		return nil, client.err
	}
	return client.result, nil
}

func llmConfig(tasks map[string]config.LLMTask, allowlist []string) config.LLM {
	return config.LLM{
		DefaultProvider: "do_gradient",
		Allowlist:       allowlist,
		Providers:       map[string]config.LLMProvider{"do_gradient": {BaseURL: "https://x", APIKey: "k"}},
		Tasks:           tasks,
	}
}

func TestCompleteReturnsValidatedJSON(t *testing.T) {
	t.Parallel()

	cfg := llmConfig(map[string]config.LLMTask{
		"parse_query": {Model: "openai-gpt-oss-20b", MaxTokens: 100, Temperature: 0.1, JSON: true, TimeoutS: 5},
	}, []string{"openai-gpt-oss-20b"})
	client := &fakeChatClient{result: &ChatResult{Content: `{"intent":"numeric","entities":[]}`, Model: "openai-gpt-oss-20b", InputTokens: 10, OutputTokens: 5}}
	service, err := New(cfg, client)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.Complete(t.Context(), "parse_query", map[string]any{"prompt": "q"})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if !result.Valid || !result.IsJSON {
		t.Fatalf("expected valid json result, got %+v", result)
	}
	if client.gotModel != "openai-gpt-oss-20b" {
		t.Fatalf("expected model routed, got %q", client.gotModel)
	}
}

func TestCompleteFlagsInvalidJSON(t *testing.T) {
	t.Parallel()

	cfg := llmConfig(map[string]config.LLMTask{
		"extract": {Model: "openai-gpt-oss-20b", JSON: true, TimeoutS: 5},
	}, []string{"openai-gpt-oss-20b"})
	client := &fakeChatClient{result: &ChatResult{Content: "not json", Model: "m"}}
	service, _ := New(cfg, client)

	result, err := service.Complete(t.Context(), "extract", nil)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid json flag")
	}
}

func TestCompleteRejectsUnknownTask(t *testing.T) {
	t.Parallel()

	service, _ := New(llmConfig(nil, nil), &fakeChatClient{})
	_, err := service.Complete(t.Context(), "bogus", nil)
	if !errors.Is(err, ErrUnknownTask) {
		t.Fatalf("expected ErrUnknownTask, got %v", err)
	}
}

func TestCompleteRejectsModelNotInAllowlist(t *testing.T) {
	t.Parallel()

	cfg := llmConfig(map[string]config.LLMTask{
		"parse_query": {Model: "openai-gpt-oss-20b", JSON: true, TimeoutS: 5},
	}, []string{"other-model"})
	service, _ := New(cfg, &fakeChatClient{})

	_, err := service.Complete(t.Context(), "parse_query", nil)
	if !errors.Is(err, ErrModelNotAllowed) {
		t.Fatalf("expected ErrModelNotAllowed, got %v", err)
	}
}

func TestCompletePropagatesEmptyResponse(t *testing.T) {
	t.Parallel()

	cfg := llmConfig(map[string]config.LLMTask{
		"parse_query": {Model: "openai-gpt-oss-20b", JSON: true, TimeoutS: 5},
	}, []string{"openai-gpt-oss-20b"})
	client := &fakeChatClient{result: &ChatResult{Content: "", Model: "m"}}
	service, _ := New(cfg, client)

	_, err := service.Complete(t.Context(), "parse_query", nil)
	if !errors.Is(err, ErrEmptyResponse) {
		t.Fatalf("expected ErrEmptyResponse, got %v", err)
	}
}

type funcChatClient struct {
	fn    func(model string) (*ChatResult, error)
	calls []string
}

func (client *funcChatClient) Complete(_ context.Context, model string, _ []Message, _ Options) (*ChatResult, error) {
	client.calls = append(client.calls, model)
	return client.fn(model)
}

func TestAllowlistFailsClosedOnEmpty(t *testing.T) {
	t.Parallel()

	cfg := llmConfig(map[string]config.LLMTask{
		"parse_query": {Model: "some-model", TimeoutS: 5},
	}, nil)
	service, _ := New(cfg, &fakeChatClient{result: &ChatResult{Content: "x", Model: "some-model"}})

	_, err := service.Complete(t.Context(), "parse_query", nil)
	if !errors.Is(err, ErrModelNotAllowed) {
		t.Fatalf("empty allowlist must fail closed, got %v", err)
	}
}

func TestFailoverToFallbackModel(t *testing.T) {
	t.Parallel()

	cfg := llmConfig(map[string]config.LLMTask{
		"synthesize_answer": {Model: "primary", FallbackModel: "backup", TimeoutS: 5},
	}, []string{"primary", "backup"})
	client := &funcChatClient{fn: func(model string) (*ChatResult, error) {
		if model == "primary" {
			return nil, errors.New("upstream unavailable")
		}
		return &ChatResult{Content: "synthesized", Model: model}, nil
	}}
	service, _ := New(cfg, client)

	result, err := service.Complete(t.Context(), "synthesize_answer", map[string]any{"prompt": "q"})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if result.Model != "backup" {
		t.Fatalf("expected failover to backup, got %q", result.Model)
	}
	if !slices.Contains(client.calls, "primary") {
		t.Fatal("primary should have been attempted before failover")
	}
}

func TestNewRequiresConfiguredProvider(t *testing.T) {
	t.Parallel()

	cfg := config.LLM{DefaultProvider: "missing", Providers: map[string]config.LLMProvider{}}
	if _, err := New(cfg, &fakeChatClient{}); !errors.Is(err, ErrProviderNotConfigured) {
		t.Fatalf("expected ErrProviderNotConfigured, got %v", err)
	}
}
