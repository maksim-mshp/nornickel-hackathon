package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	ingestpg "github.com/maksim-mshp/nornickel-hackathon/internal/ingest/adapters/pg"
	"github.com/maksim-mshp/nornickel-hackathon/internal/ingest/app"
	ingestgrpc "github.com/maksim-mshp/nornickel-hackathon/internal/ingest/ports/grpc"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/config"
	natsbus "github.com/maksim-mshp/nornickel-hackathon/internal/platform/nats"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/outbox"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/pg"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/runtime"
)

func main() {
	if err := runtime.Run("ingest", runtime.WithAssembly(buildAssembly)); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func buildAssembly(cfg config.Bundle, logger *slog.Logger) (*runtime.Assembly, error) {
	pool, err := pg.New(context.Background(), pg.Config{
		DSN:      cfg.Runtime.Postgres.DSN,
		MaxConns: cfg.Runtime.Postgres.MaxConns,
	})
	if err != nil {
		return nil, err
	}

	bus, err := natsbus.New(context.Background(), natsbus.Config{
		URL:     cfg.Runtime.NATS.URL,
		Name:    "kmap-ingest",
		Streams: streams(cfg.Runtime.NATS.Streams),
	})
	if err != nil {
		return nil, err
	}

	repository := ingestpg.NewRepository(pool.Pool)
	service := app.NewService(repository)
	grpcServer := ingestgrpc.NewServer(service)

	store := outbox.NewStore(pool.Pool)
	relay := outbox.NewRelay(store, bus, logger)

	return &runtime.Assembly{
		GRPCServices: []runtime.GRPCService{grpcServer},
		Closers:      []io.Closer{pool, bus},
		Workers:      []runtime.Worker{relay},
	}, nil
}

func streams(in []config.NATSStream) []natsbus.StreamSpec {
	out := make([]natsbus.StreamSpec, 0, len(in))
	for _, stream := range in {
		out = append(out, natsbus.StreamSpec{Name: stream.Name, Subjects: stream.Subjects})
	}
	return out
}
