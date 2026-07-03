package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	catalogpg "github.com/maksim-mshp/nornickel-hackathon/internal/catalog/adapters/pg"
	"github.com/maksim-mshp/nornickel-hackathon/internal/catalog/app"
	cataloggrpc "github.com/maksim-mshp/nornickel-hackathon/internal/catalog/ports/grpc"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/blob"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/config"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/pg"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/runtime"
)

func main() {
	if err := runtime.Run("catalog", runtime.WithAssembly(build)); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func build(cfg config.Bundle, _ *slog.Logger) (*runtime.Assembly, error) {
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

	service := app.NewService(catalogpg.NewRepo(pool.Pool), store)
	return &runtime.Assembly{
		GRPCServices: []runtime.GRPCService{cataloggrpc.NewServer(service)},
		Closers:      []io.Closer{pool},
	}, nil
}
