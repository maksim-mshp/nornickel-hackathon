package consumer

import (
	"context"
	"log/slog"
	"testing"

	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/events"
)

type fakeBus struct {
	published []events.Envelope
}

func (bus *fakeBus) Publish(_ context.Context, env events.Envelope) error {
	bus.published = append(bus.published, env)
	return nil
}

func (bus *fakeBus) Subscribe(context.Context, events.Subscription) error {
	return nil
}

type fakeRecalculator struct {
	factIDs []string
}

func (service *fakeRecalculator) RecalculateFacts(_ context.Context, factIDs []string) ([]string, error) {
	service.factIDs = factIDs
	return []string{"cluster-1"}, nil
}

func TestHandleFactsCommittedPublishesEpistemicUpdated(t *testing.T) {
	t.Parallel()

	bus := &fakeBus{}
	service := &fakeRecalculator{}
	worker := NewWorker(bus, service, slog.Default())
	env, err := events.New(events.Event{
		Type:    events.FactsCommitted,
		Source:  "kmap/catalog",
		Subject: "doc-1",
		Data: map[string]any{
			"document_id": "doc-1",
			"fact_ids":    []string{"fact-1"},
		},
	})
	if err != nil {
		t.Fatalf("build event: %v", err)
	}

	action := worker.handle(t.Context(), events.Message{Envelope: env})

	if action != events.Ack {
		t.Fatalf("expected ack, got %v", action)
	}
	if len(service.factIDs) != 1 || service.factIDs[0] != "fact-1" {
		t.Fatalf("unexpected fact ids: %v", service.factIDs)
	}
	if len(bus.published) != 1 {
		t.Fatalf("expected one published event, got %d", len(bus.published))
	}
	if bus.published[0].Type != events.EpistemicUpdated {
		t.Fatalf("expected epistemic updated, got %q", bus.published[0].Type)
	}
}
