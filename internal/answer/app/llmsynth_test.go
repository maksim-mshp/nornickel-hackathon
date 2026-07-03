package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

type fakeLLM struct {
	lastReq *kmapv1.CompleteRequest
	content string
	err     error
}

func (f *fakeLLM) Complete(_ context.Context, in *kmapv1.CompleteRequest, _ ...grpc.CallOption) (*kmapv1.CompleteResponse, error) {
	f.lastReq = in
	if f.err != nil {
		return nil, f.err
	}
	js, _ := structpb.NewStruct(map[string]any{"text": f.content})
	return &kmapv1.CompleteResponse{Json: js, Model: "test-model"}, nil
}

func TestLLMSynthesizerRendersEvidenceAndParsesResponse(t *testing.T) {
	t.Parallel()
	pack := samplecatholytePack(t)
	llm := &fakeLLM{content: "Скорость циркуляции католита — 0,8–1 м/с [F1]."}
	synth := NewLLMSynthesizer(llm)

	result, err := synth.Synthesize(context.Background(), "оптимальная скорость католита?", pack)
	if err != nil {
		t.Fatalf("synthesize: %v", err)
	}
	if result.Summary != llm.content {
		t.Fatalf("want llm content, got %q", result.Summary)
	}
	if llm.lastReq.GetTask() != "synthesize_answer" {
		t.Fatalf("want task synthesize_answer, got %q", llm.lastReq.GetTask())
	}
	messages := llm.lastReq.GetPayload().GetFields()["messages"].GetListValue()
	if messages == nil || len(messages.GetValues()) != 2 {
		t.Fatal("want system and user messages in payload")
	}
	user := messages.GetValues()[1].GetStructValue().GetFields()["content"].GetStringValue()
	if !strings.Contains(user, "[F1]") || !strings.Contains(user, "оптимальная скорость католита?") {
		t.Fatalf("user prompt must carry evidence refs and question: %q", user)
	}
}

func TestLLMSynthesizerErrorsOnEmptyContent(t *testing.T) {
	t.Parallel()
	if _, err := NewLLMSynthesizer(&fakeLLM{content: ""}).Synthesize(context.Background(), "q", samplecatholytePack(t)); err == nil {
		t.Fatal("want error on empty llm content")
	}
}

func TestLLMSynthesizerPropagatesError(t *testing.T) {
	t.Parallel()
	if _, err := NewLLMSynthesizer(&fakeLLM{err: errors.New("upstream down")}).Synthesize(context.Background(), "q", samplecatholytePack(t)); err == nil {
		t.Fatal("want error propagated from llm client")
	}
}
