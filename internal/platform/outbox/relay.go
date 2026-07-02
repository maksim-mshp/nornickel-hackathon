package outbox

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/events"
)

type Relay struct {
	store     Store
	publisher events.Publisher
	logger    *slog.Logger
	batch     int
	interval  time.Duration
}

type Option func(*Relay)

func NewRelay(store Store, publisher events.Publisher, logger *slog.Logger, options ...Option) *Relay {
	relay := &Relay{
		store:     store,
		publisher: publisher,
		logger:    logger,
		batch:     DefaultBatch,
		interval:  DefaultInterval,
	}
	for _, option := range options {
		option(relay)
	}
	return relay
}

func WithBatch(size int) Option {
	return func(relay *Relay) {
		if size > 0 {
			relay.batch = size
		}
	}
}

func WithInterval(interval time.Duration) Option {
	return func(relay *Relay) {
		if interval > 0 {
			relay.interval = interval
		}
	}
}

func (relay *Relay) Run(ctx context.Context) error {
	ticker := time.NewTicker(relay.interval)
	defer ticker.Stop()

	for {
		if err := relay.drain(ctx); err != nil && !errors.Is(err, context.Canceled) {
			relay.logger.Error("outbox drain failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (relay *Relay) drain(ctx context.Context) error {
	records, err := relay.store.Claim(ctx, relay.batch)
	if err != nil {
		return err
	}
	for _, record := range records {
		if err := relay.publisher.Publish(ctx, record.Envelope); err != nil {
			return err
		}
		id, parseErr := uuid.Parse(record.Envelope.ID)
		if parseErr != nil {
			relay.logger.Error("outbox envelope id is not uuid", "id", record.Envelope.ID, "error", parseErr)
			continue
		}
		if err := relay.store.MarkPublished(ctx, id); err != nil {
			return err
		}
	}
	if len(records) > 0 {
		relay.logger.Info("outbox drained", "published", len(records))
	}
	return nil
}
