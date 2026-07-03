package app

import (
	"context"
	"strings"
	"testing"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

type fakeSearch struct {
	pack *kmapv1.EvidencePack
}

func (f fakeSearch) Search(context.Context, *kmapv1.SearchRequest, ...grpc.CallOption) (*kmapv1.SearchResponse, error) {
	return &kmapv1.SearchResponse{Evidence: f.pack}, nil
}

func TestAskEmitsOrderedEventsAndPassesGuard(t *testing.T) {
	t.Parallel()

	service := NewService(fakeSearch{pack: samplecatholytePack(t)})
	var events []*kmapv1.AskResponse
	err := service.Ask(context.Background(), &kmapv1.AskRequest{Question: "оптимальная скорость циркуляции католита при электроэкстракции никеля?"}, func(event *kmapv1.AskResponse) error {
		events = append(events, event)
		return nil
	})
	if err != nil {
		t.Fatalf("ask failed: %v", err)
	}

	if len(events) < 4 {
		t.Fatalf("expected at least 4 events, got %d", len(events))
	}
	if events[0].GetType() != "plan" {
		t.Fatalf("expected first event plan, got %s", events[0].GetType())
	}
	if events[1].GetType() != "evidence" {
		t.Fatalf("expected second event evidence, got %s", events[1].GetType())
	}

	var summary strings.Builder
	var done *kmapv1.AskResponse
	for _, event := range events {
		switch event.GetType() {
		case "answer.delta":
			summary.WriteString(event.GetDelta())
		case "answer.done":
			done = event
		}
	}

	if done == nil {
		t.Fatal("expected answer.done event")
	}
	if done.GetAnswer().GetSummary() != summary.String() {
		t.Fatalf("streamed deltas must equal final summary\ndeltas: %q\nfinal:  %q", summary.String(), done.GetAnswer().GetSummary())
	}
	if done.GetAnswer().GetGuard().GetNumbersChecked() == 0 {
		t.Fatal("expected guard to check numbers")
	}
	if done.GetAnswer().GetGuard().GetViolations() != 0 {
		t.Fatalf("expected zero guard violations, got %d", done.GetAnswer().GetGuard().GetViolations())
	}
	if !strings.Contains(summary.String(), "[F1]") {
		t.Fatalf("expected citation in summary: %q", summary.String())
	}
}

type hallucinatingSynth struct{}

func (hallucinatingSynth) Synthesize(context.Context, string, *kmapv1.EvidencePack) (Synthesis, error) {
	return Synthesis{Summary: "Скорость составляет 999,9 м/с по всем источникам.", Confidence: 0.9}, nil
}

func TestAskDegradesOnGuardViolation(t *testing.T) {
	t.Parallel()

	service := NewService(fakeSearch{pack: samplecatholytePack(t)}, WithSynthesizer(hallucinatingSynth{}))
	var done *kmapv1.AskResponse
	err := service.Ask(context.Background(), &kmapv1.AskRequest{Question: "скорость католита?"}, func(event *kmapv1.AskResponse) error {
		if event.GetType() == "answer.done" {
			done = event
		}
		return nil
	})
	if err != nil {
		t.Fatalf("ask failed: %v", err)
	}
	if done == nil {
		t.Fatal("expected answer.done event")
	}
	if !done.GetAnswer().GetGuard().GetDegraded() {
		t.Fatal("expected guard to degrade on hallucinated number")
	}
	if done.GetAnswer().GetGuard().GetViolations() != 0 {
		t.Fatalf("expected zero violations after degradation, got %d", done.GetAnswer().GetGuard().GetViolations())
	}
	if strings.Contains(done.GetAnswer().GetSummary(), "999") {
		t.Fatalf("degraded answer must not contain hallucinated number: %q", done.GetAnswer().GetSummary())
	}
}

func TestGuardFlagsForeignNumber(t *testing.T) {
	t.Parallel()

	pack := samplecatholytePack(t)
	if got := runGuard("значение 999,9 м/с не из источников", pack); got.violations == 0 {
		t.Fatalf("expected guard violation for foreign number, got %+v", got)
	}
}

func samplecatholytePack(t *testing.T) *kmapv1.EvidencePack {
	t.Helper()
	fact := mustStruct(t, map[string]any{
		"ref":        "F1",
		"subject":    map[string]any{"slug": "process:nickel-electrowinning", "name": "электроэкстракция никеля"},
		"parameter":  map[string]any{"slug": "parameter:catholyte-flow-rate", "name": "скорость циркуляции католита"},
		"value":      map[string]any{"operator": "range", "vmin": 0.8, "vmax": 1.0, "unit": "м/с"},
		"si":         map[string]any{"operator": "range", "vmin": 0.8, "vmax": 1.0, "unit": "m/s"},
		"conditions": map[string]any{"плотность тока": "220 А/м²"},
		"geography":  "foreign",
	})
	consensus := mustStruct(t, map[string]any{
		"parameter": map[string]any{"slug": "parameter:catholyte-flow-rate", "name": "скорость циркуляции католита"},
		"unit":      "м/с",
		"verdict":   "majority",
		"agreedMin": 0.8,
		"agreedMax": 0.9,
	})
	return &kmapv1.EvidencePack{
		Facts:     []*kmapv1.Fact{{Id: "f1", Kind: "numeric", Payload: fact}},
		Consensus: []*structpb.Struct{consensus},
	}
}

func mustStruct(t *testing.T, value map[string]any) *structpb.Struct {
	t.Helper()
	result, err := structpb.NewStruct(value)
	if err != nil {
		t.Fatalf("build struct: %v", err)
	}
	return result
}
