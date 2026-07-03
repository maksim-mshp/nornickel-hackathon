package outbox

import (
	"context"
	"errors"
	"log/slog"
	"time"

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

type headerPublisher interface {
	PublishWithHeaders(ctx context.Context, env events.Envelope, headers map[string]string) error
}

func (relay *Relay) publish(ctx context.Context, record Record) error {
	if hp, ok := relay.publisher.(headerPublisher); ok && len(record.Headers) > 0 {
		return hp.PublishWithHeaders(ctx, record.Envelope, record.Headers)
	}
	return relay.publisher.Publish(ctx, record.Envelope)
}

func (relay *Relay) drain(ctx context.Context) error {
	published, err := relay.store.Drain(ctx, relay.batch, func(ctx context.Context, record Record) error {
		if publishErr := relay.publish(ctx, record); publishErr != nil {
			relay.logger.Warn("outbox publish failed", "event", record.Envelope.Type, "error", publishErr)
			return publishErr
		}
		return nil
	})
	if err != nil {
		return err
	}
	if published > 0 {
		relay.logger.Info("outbox drained", "published", published)
	}
	return nil
}
