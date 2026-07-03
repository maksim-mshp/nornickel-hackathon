package openai

import (
	"context"
	"os"
	"testing"

	"github.com/maksim-mshp/nornickel-hackathon/internal/llm/app"
)

func TestLiveComplete(t *testing.T) {
	key := os.Getenv("KMAP_YANDEX_KEY")
	model := os.Getenv("KMAP_YANDEX_MODEL")
	if key == "" || model == "" {
		t.Skip("set KMAP_YANDEX_KEY and KMAP_YANDEX_MODEL to run the live Yandex test")
	}

	client := New("https://llm.api.cloud.yandex.net/v1", key, "api-key")
	result, err := client.Complete(context.Background(), model, []app.Message{
		{Role: "system", Content: "Отвечай кратко."},
		{Role: "user", Content: "Ответь одним словом: столица России?"},
	}, app.Options{Temperature: 0.1, MaxTokens: 20})
	if err != nil {
		t.Fatalf("live complete: %v", err)
	}
	if result.Content == "" {
		t.Fatal("empty content from upstream")
	}
	t.Logf("content=%q model=%s tokens_in=%d tokens_out=%d", result.Content, result.Model, result.InputTokens, result.OutputTokens)
}
