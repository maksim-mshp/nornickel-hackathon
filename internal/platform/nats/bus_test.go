package nats_test

import (
	"context"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/events"
	natsbus "github.com/maksim-mshp/nornickel-hackathon/internal/platform/nats"
)

func startJetStream(t *testing.T) string {
	t.Helper()

	opts := &natsserver.Options{
		JetStream: true,
		StoreDir:  t.TempDir(),
		Host:      "127.0.0.1",
		Port:      -1,
		NoLog:     true,
		NoSigs:    true,
	}
	server, err := natsserver.NewServer(opts)
	if err != nil {
		t.Fatalf("create nats server: %v", err)
	}
	go server.Start()
	if !server.ReadyForConnections(2 * time.Second) {
		t.Fatal("nats server not ready")
	}
	t.Cleanup(server.Shutdown)
	return server.ClientURL()
}

func TestBusPublishSubscribeRoundTrip(t *testing.T) {
	t.Parallel()

	url := startJetStream(t)
	ctx := t.Context()

	bus, err := natsbus.New(ctx, natsbus.Config{URL: url, Streams: natsbus.DefaultStreams()})
	if err != nil {
		t.Fatalf("create bus: %v", err)
	}
	t.Cleanup(func() { _ = bus.Close() })

	received := make(chan events.Envelope, 1)
	if err := bus.Subscribe(ctx, events.Subscription{
		Subject: events.DocumentRegistered,
		Durable: "test-roundtrip",
		Handler: func(_ context.Context, msg events.Message) events.AckAction {
			received <- msg.Envelope
			return events.Ack
		},
	}); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	env, err := events.New(events.Event{
		Type:    events.DocumentRegistered,
		Source:  "kmap/ingest",
		Subject: "doc-1",
		Data:    map[string]any{"sha256": "ab"},
	})
	if err != nil {
		t.Fatalf("new event: %v", err)
	}
	if err := bus.Publish(ctx, env); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case got := <-received:
		if got.ID != env.ID {
			t.Fatalf("expected id %s, got %s", env.ID, got.ID)
		}
		if got.Type != env.Type {
			t.Fatalf("expected type %s, got %s", env.Type, got.Type)
		}
		var data map[string]any
		if err := got.UnmarshalData(&data); err != nil {
			t.Fatalf("unmarshal data: %v", err)
		}
		if data["sha256"] != "ab" {
			t.Fatalf("unexpected data: %v", data)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestBusDeduplicatesByMsgID(t *testing.T) {
	t.Parallel()

	url := startJetStream(t)
	ctx := t.Context()

	bus, err := natsbus.New(ctx, natsbus.Config{URL: url, Streams: natsbus.DefaultStreams()})
	if err != nil {
		t.Fatalf("create bus: %v", err)
	}
	t.Cleanup(func() { _ = bus.Close() })

	received := make(chan events.Envelope, 8)
	if err := bus.Subscribe(ctx, events.Subscription{
		Subject: events.DocumentParsed,
		Durable: "test-dedup",
		Handler: func(_ context.Context, msg events.Message) events.AckAction {
			received <- msg.Envelope
			return events.Ack
		},
	}); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	env, err := events.New(events.Event{
		Type:    events.DocumentParsed,
		Source:  "kmap/parse",
		Subject: "doc-2",
		Data:    map[string]any{"pages": 10},
	})
	if err != nil {
		t.Fatalf("new event: %v", err)
	}

	for range 2 {
		if err := bus.Publish(ctx, env); err != nil {
			t.Fatalf("publish: %v", err)
		}
	}

	count := 0
	timer := time.After(time.Second)
loop:
	for {
		select {
		case <-received:
			count++
		case <-timer:
			break loop
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 delivery (dedup by nats-msg-id), got %d", count)
	}
}
