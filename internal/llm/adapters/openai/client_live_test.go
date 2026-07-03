package openai

import (
	"context"
	"os"
	"testing"

	"github.com/maksim-mshp/nornickel-hackathon/internal/llm/app"
)

func TestLiveComplete(t *testing.T) {
	key := os.Getenv("KMAP_YANDEX_KEY")
	folder := os.Getenv("KMAP_YANDEX_FOLDER")
	model := os.Getenv("KMAP_YANDEX_MODEL")
	if key == "" || folder == "" || model == "" {
		t.Skip("set KMAP_YANDEX_KEY, KMAP_YANDEX_FOLDER and KMAP_YANDEX_MODEL to run the live Yandex test")
	}

	client := New("https://ai.api.cloud.yandex.net/v1", key, "api-key", folder)
	result, err := client.Complete(context.Background(), model, []app.Message{
		{Role: "system", Content: "Отвечай кратко."},
		{Role: "user", Content: "Ответь одним словом: столица России?"},
	}, app.Options{Temperature: 0.1, MaxTokens: 2000})
	if err != nil {
		t.Fatalf("live complete: %v", err)
	}
	if result.Content == "" {
		t.Fatal("empty content from upstream")
	}
	t.Logf("content=%q model=%s tokens_in=%d tokens_out=%d", result.Content, result.Model, result.InputTokens, result.OutputTokens)
}
