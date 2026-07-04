package openai

import (
	"context"
	"errors"
	"testing"

	"github.com/maksim-mshp/nornickel-hackathon/internal/llm/app"
)

type stubClient struct {
	reply string
	err   error
	calls []string
}

func (s *stubClient) Complete(_ context.Context, model string, _ []app.Message, _ app.Options) (*app.ChatResult, error) {
	s.calls = append(s.calls, model)
	if s.err != nil {
		return nil, s.err
	}
	return &app.ChatResult{Content: s.reply, Model: model}, nil
}

func TestFallbackUsesPrimaryWhenHealthy(t *testing.T) {
	t.Parallel()

	primary := &stubClient{reply: "yandex"}
	secondary := &stubClient{reply: "do"}
	client, err := NewFallback(nil,
		Provider{Name: "yandex", Client: primary},
		Provider{Name: "do_gradient", Client: secondary, Models: map[string]string{"gpt-oss-20b/latest": "openai-gpt-oss-20b"}},
	)
	if err != nil {
		t.Fatalf("new fallback: %v", err)
	}

	result, err := client.Complete(t.Context(), "gpt-oss-20b/latest", nil, app.Options{})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if result.Model != "gpt-oss-20b/latest" {
		t.Fatalf("expected primary model, got %q", result.Model)
	}
	if len(secondary.calls) != 0 {
		t.Fatalf("secondary must not be called when primary succeeds, got %v", secondary.calls)
	}
}

func TestFallbackTranslatesModelAndFailsOver(t *testing.T) {
	t.Parallel()

	primary := &stubClient{err: errors.New("permission denied")}
	secondary := &stubClient{reply: "do"}
	client, _ := NewFallback(nil,
		Provider{Name: "yandex", Client: primary},
		Provider{Name: "do_gradient", Client: secondary, Models: map[string]string{"gpt-oss-20b/latest": "openai-gpt-oss-20b"}},
	)

	result, err := client.Complete(t.Context(), "gpt-oss-20b/latest", nil, app.Options{})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if result.Model != "openai-gpt-oss-20b" {
		t.Fatalf("expected translated do model, got %q", result.Model)
	}
	if len(primary.calls) != 1 || primary.calls[0] != "gpt-oss-20b/latest" {
		t.Fatalf("primary should receive canonical model, got %v", primary.calls)
	}
}

func TestFallbackSkipsProviderWithoutModelMapping(t *testing.T) {
	t.Parallel()

	primary := &stubClient{err: errors.New("down")}
	secondary := &stubClient{reply: "do"}
	client, _ := NewFallback(nil,
		Provider{Name: "yandex", Client: primary},
		Provider{Name: "do_gradient", Client: secondary, Models: map[string]string{"other/latest": "x"}},
	)

	_, err := client.Complete(t.Context(), "gpt-oss-20b/latest", nil, app.Options{})
	if err == nil {
		t.Fatal("expected error when no provider serves the model")
	}
	if len(secondary.calls) != 0 {
		t.Fatalf("secondary lacks mapping and must be skipped, got %v", secondary.calls)
	}
}

func TestNewFallbackRejectsEmptyChain(t *testing.T) {
	t.Parallel()

	if _, err := NewFallback(nil); err == nil {
		t.Fatal("expected error for empty provider chain")
	}
}
