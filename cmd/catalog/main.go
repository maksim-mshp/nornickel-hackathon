package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	catalogpg "github.com/maksim-mshp/nornickel-hackathon/internal/catalog/adapters/pg"
	"github.com/maksim-mshp/nornickel-hackathon/internal/catalog/app"
	catalogconsumer "github.com/maksim-mshp/nornickel-hackathon/internal/catalog/ports/consumer"
	cataloggrpc "github.com/maksim-mshp/nornickel-hackathon/internal/catalog/ports/grpc"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/blob"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/config"
	natsbus "github.com/maksim-mshp/nornickel-hackathon/internal/platform/nats"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/pg"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/runtime"
)

func main() {
	if err := runtime.Run("catalog", runtime.WithAssembly(build)); err != nil {
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

	store, err := blob.New(blob.Config{
		Endpoint:  cfg.Runtime.S3.Endpoint,
		AccessKey: cfg.Runtime.S3.AccessKey,
		SecretKey: cfg.Runtime.S3.SecretKey,
		UseSSL:    cfg.Runtime.S3.UseSSL,
		Region:    cfg.Runtime.S3.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create s3 client: %w", err)
	}

	bus, err := natsbus.New(context.Background(), natsbus.Config{
		URL:     cfg.Runtime.NATS.URL,
		Name:    "kmap-catalog",
		Streams: streams(cfg.Runtime.NATS.Streams),
	})
	if err != nil {
		return nil, err
	}

	service := app.NewService(catalogpg.NewRepo(pool.Pool), store)
	worker := catalogconsumer.NewWorker(bus, service, logger)
	return &runtime.Assembly{
		GRPCServices: []runtime.GRPCService{cataloggrpc.NewServer(service)},
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
