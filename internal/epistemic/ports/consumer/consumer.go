package consumer

import (
	"context"
	"log/slog"

	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/events"
)

const eventSource = "kmap/epistemic"

type Recalculator interface {
	RecalculateFacts(ctx context.Context, factIDs []string) ([]string, error)
}

type Bus interface {
	events.Bus
}

type Worker struct {
	bus     Bus
	service Recalculator
	logger  *slog.Logger
}

func NewWorker(bus Bus, service Recalculator, logger *slog.Logger) *Worker {
	return &Worker{bus: bus, service: service, logger: logger}
}

func (worker *Worker) Run(ctx context.Context) error {
	if err := worker.bus.Subscribe(ctx, events.Subscription{
		Subject: events.FactsCommitted,
		Durable: "kmap-epistemic-facts",
		Handler: worker.handle,
	}); err != nil {
		return err
	}
	<-ctx.Done()
	return nil
}

func (worker *Worker) handle(ctx context.Context, msg events.Message) events.AckAction {
	var data struct {
		DocumentID  string   `json:"document_id"`
		FactIDs     []string `json:"fact_ids"`
		ClusterKeys []string `json:"cluster_keys"`
	}
	if err := msg.Envelope.UnmarshalData(&data); err != nil {
		worker.logger.Warn("skip facts event", "error", err)
		return events.Term
	}
	clusterKeys, err := worker.service.RecalculateFacts(ctx, data.FactIDs)
	if err != nil {
		worker.logger.Error("epistemic recalculation failed", "document_id", data.DocumentID, "error", err)
		return events.Nack
	}
	if len(clusterKeys) == 0 {
		clusterKeys = data.ClusterKeys
	}
	env, err := events.New(events.Event{
		Type:    events.EpistemicUpdated,
		Source:  eventSource,
		Subject: data.DocumentID,
		Data: map[string]any{
			"document_id":  data.DocumentID,
			"cluster_keys": clusterKeys,
		},
	})
	if err != nil {
		worker.logger.Error("build epistemic updated event failed", "document_id", data.DocumentID, "error", err)
		return events.Nack
	}
	if err := worker.bus.Publish(ctx, env); err != nil {
		worker.logger.Error("publish epistemic updated failed", "document_id", data.DocumentID, "error", err)
		return events.Nack
	}
	worker.logger.Info("epistemic recalculated", "document_id", data.DocumentID, "clusters", len(clusterKeys))
	return events.Ack
}
