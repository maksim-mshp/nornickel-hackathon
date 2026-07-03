package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	epistemicpg "github.com/maksim-mshp/nornickel-hackathon/internal/epistemic/adapters/pg"
	"github.com/maksim-mshp/nornickel-hackathon/internal/epistemic/app"
	epistemicconsumer "github.com/maksim-mshp/nornickel-hackathon/internal/epistemic/ports/consumer"
	epistemicgrpc "github.com/maksim-mshp/nornickel-hackathon/internal/epistemic/ports/grpc"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/config"
	natsbus "github.com/maksim-mshp/nornickel-hackathon/internal/platform/nats"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/pg"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/runtime"
)

func main() {
	if err := runtime.Run("epistemic", runtime.WithAssembly(build)); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func build(cfg config.Bundle, logger *slog.Logger) (*runtime.Assembly, error) {
	pool, err := pg.New(context.Background(), pg.Config{
		DSN:      cfg.Runtime.Postgres.DSN,
		MaxConns: cfg.Runtime.Postgres.MaxConns,
	})
	if err != nil {
		return nil, err
	}

	bus, err := natsbus.New(context.Background(), natsbus.Config{
		URL:     cfg.Runtime.NATS.URL,
		Name:    "kmap-epistemic",
		Streams: streams(cfg.Runtime.NATS.Streams),
	})
	if err != nil {
		return nil, err
	}

	service := app.NewService(epistemicpg.NewRepo(pool.Pool))
	worker := epistemicconsumer.NewWorker(bus, service, logger)
	return &runtime.Assembly{
		GRPCServices: []runtime.GRPCService{epistemicgrpc.NewServer(service)},
		Closers:      []io.Closer{pool, bus},
		Workers:      []runtime.Worker{worker},
	}, nil
}

func streams(in []config.NATSStream) []natsbus.StreamSpec {
	out := make([]natsbus.StreamSpec, 0, len(in))
	for _, stream := range in {
		out = append(out, natsbus.StreamSpec{Name: stream.Name, Subjects: stream.Subjects})
	}
	return out
}
