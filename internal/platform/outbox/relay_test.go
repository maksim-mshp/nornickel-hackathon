package outbox

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/events"
)

type fakeStore struct {
	mu       sync.Mutex
	records  []Record
	marked   []string
	claimErr error
}

func (store *fakeStore) Drain(ctx context.Context, _ int, publish func(context.Context, Record) error) (int, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.claimErr != nil {
		return 0, store.claimErr
	}
	published := 0
	for _, record := range store.records {
		if err := publish(ctx, record); err != nil {
			continue
		}
		store.marked = append(store.marked, record.Envelope.ID)
		published++
	}
	return published, nil
}

type fakePublisher struct {
	mu        sync.Mutex
	published []events.Envelope
	err       error
}

func (publisher *fakePublisher) Publish(_ context.Context, env events.Envelope) error {
	publisher.mu.Lock()
	defer publisher.mu.Unlock()
	if publisher.err != nil {
		return publisher.err
	}
	publisher.published = append(publisher.published, env)
	return nil
}

type fakeHeaderPublisher struct {
	fakePublisher
	headers []map[string]string
}

func (publisher *fakeHeaderPublisher) PublishWithHeaders(_ context.Context, env events.Envelope, headers map[string]string) error {
	publisher.mu.Lock()
	defer publisher.mu.Unlock()
	publisher.published = append(publisher.published, env)
	publisher.headers = append(publisher.headers, headers)
	return nil
}

func TestRelayForwardsHeadersWhenSupported(t *testing.T) {
	t.Parallel()

	record := newTestRecord(t, "a")
	record.Headers = map[string]string{"traceparent": "00-trace-span-01"}
	store := &fakeStore{records: []Record{record}}
	publisher := &fakeHeaderPublisher{}
	relay := NewRelay(store, publisher, slog.Default())

	if err := relay.drain(context.Background()); err != nil {
		t.Fatalf("drain failed: %v", err)
	}
	if len(publisher.headers) != 1 || publisher.headers[0]["traceparent"] != "00-trace-span-01" {
		t.Fatalf("expected traceparent forwarded to broker, got %v", publisher.headers)
	}
}

func newTestRecord(t *testing.T, subject string) Record {
	t.Helper()
	env, err := events.New(events.Event{
		Type:    events.DocumentRegistered,
		Source:  "kmap/ingest",
		Subject: subject,
		Data:    map[string]any{"x": 1},
	})
	if err != nil {
		t.Fatalf("new envelope: %v", err)
	}
	return Record{Envelope: env, AggregateType: "document"}
}

func TestRelayDrainPublishesAndMarks(t *testing.T) {
	t.Parallel()

	store := &fakeStore{records: []Record{newTestRecord(t, "a"), newTestRecord(t, "b")}}
	publisher := &fakePublisher{}
	relay := NewRelay(store, publisher, slog.Default())

	if err := relay.drain(context.Background()); err != nil {
		t.Fatalf("expected drain to succeed: %v", err)
	}
	if len(publisher.published) != 2 {
		t.Fatalf("expected 2 published, got %d", len(publisher.published))
	}
	if len(store.marked) != 2 {
		t.Fatalf("expected 2 marked, got %d", len(store.marked))
	}
}

func TestRelayDrainSkipsFailedPublishWithoutMarking(t *testing.T) {
	t.Parallel()

	store := &fakeStore{records: []Record{newTestRecord(t, "a"), newTestRecord(t, "b")}}
	publisher := &fakePublisher{err: errors.New("nats down")}
	relay := NewRelay(store, publisher, slog.Default())

	if err := relay.drain(context.Background()); err != nil {
		t.Fatalf("expected drain to tolerate publish errors, got %v", err)
	}
	if len(publisher.published) != 0 {
		t.Fatalf("expected 0 published, got %d", len(publisher.published))
	}
	if len(store.marked) != 0 {
		t.Fatalf("expected 0 marked (failed events stay for retry), got %d", len(store.marked))
	}
}

func TestRelayDrainPropagatesClaimError(t *testing.T) {
	t.Parallel()

	store := &fakeStore{claimErr: errors.New("db down")}
	publisher := &fakePublisher{}
	relay := NewRelay(store, publisher, slog.Default())

	if err := relay.drain(context.Background()); err == nil {
		t.Fatal("expected drain to fail on claim error")
	}
}

func TestRelayRunStopsOnContextCancel(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	publisher := &fakePublisher{}
	relay := NewRelay(store, publisher, slog.Default(), WithInterval(10*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- relay.Run(ctx) }()

	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected Run to return context error")
		}
	case <-time.After(time.Second):
		t.Fatal("relay did not stop after context cancel")
	}
}

func TestOptionsApply(t *testing.T) {
	t.Parallel()

	relay := NewRelay(&fakeStore{}, &fakePublisher{}, slog.Default(),
		WithBatch(7), WithInterval(250*time.Millisecond))
	if relay.batch != 7 {
		t.Fatalf("expected batch 7, got %d", relay.batch)
	}
	if relay.interval != 250*time.Millisecond {
		t.Fatalf("expected interval 250ms, got %v", relay.interval)
	}
}
