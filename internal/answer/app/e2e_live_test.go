package app

import (
	"context"
	"os"
	"testing"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestLiveLLMSynthesisPassesGuard(t *testing.T) {
	addr := os.Getenv("KMAP_LLM_ADDR")
	if addr == "" {
		t.Skip("set KMAP_LLM_ADDR (e.g. localhost:9095) to run the live end-to-end LLM synthesis test against a running kmap-llm")
	}

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial llm: %v", err)
	}
	defer func() { _ = conn.Close() }()

	pack := samplecatholytePack(t)
	synth := NewLLMSynthesizer(kmapv1.NewLLMServiceClient(conn))
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
