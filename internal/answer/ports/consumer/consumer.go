package consumer

import (
	"context"
	"log/slog"

	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/events"
)

type Invalidator interface {
	ResolveSlugs(ctx context.Context, entityIDs []string) ([]string, error)
	Invalidate(ctx context.Context, slugs []string) (int64, error)
}

type Bus interface {
	Subscribe(ctx context.Context, sub events.Subscription) error
}

type Worker struct {
	bus    Bus
	cache  Invalidator
	logger *slog.Logger
}

func NewWorker(bus Bus, cache Invalidator, logger *slog.Logger) *Worker {
	return &Worker{bus: bus, cache: cache, logger: logger}
}

func (worker *Worker) Run(ctx context.Context) error {
	if err := worker.bus.Subscribe(ctx, events.Subscription{
		Subject: events.FactsCommitted,
		Durable: "kmap-answer-cache-invalidate",
		Handler: worker.handle,
	}); err != nil {
		return err
	}
	<-ctx.Done()
	return nil
}

func (worker *Worker) handle(ctx context.Context, msg events.Message) events.AckAction {
	var data struct {
		EntityIDs []string `json:"entity_ids"`
	}
	if err := msg.Envelope.UnmarshalData(&data); err != nil {
		worker.logger.Warn("skip facts event for cache invalidation", "error", err)
		return events.Term
	}
	if len(data.EntityIDs) == 0 {
		return events.Ack
	}
	slugs, err := worker.cache.ResolveSlugs(ctx, data.EntityIDs)
	if err != nil {
		worker.logger.Error("resolve slugs for invalidation failed", "error", err)
		return events.Nack
	}
	removed, err := worker.cache.Invalidate(ctx, slugs)
	if err != nil {
		worker.logger.Error("cache invalidation failed", "error", err)
		return events.Nack
	}
	if removed > 0 {
		worker.logger.Info("answer cache invalidated", "entries", removed)
	}
	return events.Ack
}
