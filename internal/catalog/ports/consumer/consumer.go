package consumer

import (
	"context"
	"log/slog"

	"github.com/maksim-mshp/nornickel-hackathon/internal/catalog/app"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/events"
)

type Committer interface {
	CommitExtraction(ctx context.Context, bundleURI string) (app.CommitResult, error)
}

type Bus interface {
	Subscribe(ctx context.Context, sub events.Subscription) error
}

type Worker struct {
	bus     Bus
	service Committer
	logger  *slog.Logger
}

func NewWorker(bus Bus, service Committer, logger *slog.Logger) *Worker {
	return &Worker{bus: bus, service: service, logger: logger}
}

func (worker *Worker) Run(ctx context.Context) error {
	if err := worker.bus.Subscribe(ctx, events.Subscription{
		Subject: events.DocumentExtracted,
		Durable: "kmap-catalog-extracted",
		Handler: worker.handle,
	}); err != nil {
		return err
	}
	<-ctx.Done()
	return nil
}

func (worker *Worker) handle(ctx context.Context, msg events.Message) events.AckAction {
	var data struct {
		DocumentID string `json:"document_id"`
		BundleURI  string `json:"bundle_uri"`
	}
	if err := msg.Envelope.UnmarshalData(&data); err != nil || data.BundleURI == "" {
		worker.logger.Warn("skip extracted event", "error", err)
		return events.Term
	}
	result, err := worker.service.CommitExtraction(ctx, data.BundleURI)
	if err != nil {
		worker.logger.Error("commit extraction failed", "document_id", data.DocumentID, "error", err)
		return events.Nack
	}
	worker.logger.Info("committed extraction", "document_id", data.DocumentID, "facts", len(result.FactIDs))
	return events.Ack
}
