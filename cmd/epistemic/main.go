package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	epistemicpg "github.com/maksim-mshp/nornickel-hackathon/internal/epistemic/adapters/pg"
	"github.com/maksim-mshp/nornickel-hackathon/internal/epistemic/app"
	epistemicgrpc "github.com/maksim-mshp/nornickel-hackathon/internal/epistemic/ports/grpc"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/config"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/pg"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/runtime"
)

func main() {
	if err := runtime.Run("epistemic", runtime.WithAssembly(build)); err != nil {
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

	service := app.NewService(epistemicpg.NewRepo(pool.Pool))
	return &runtime.Assembly{
		GRPCServices: []runtime.GRPCService{epistemicgrpc.NewServer(service)},
		Closers:      []io.Closer{pool},
	}, nil
}
