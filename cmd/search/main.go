package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/config"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/pg"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/runtime"
	searchpg "github.com/maksim-mshp/nornickel-hackathon/internal/search/adapters/pg"
	"github.com/maksim-mshp/nornickel-hackathon/internal/search/app"
	searchgrpc "github.com/maksim-mshp/nornickel-hackathon/internal/search/ports/grpc"
)

const currentYear = 2026

func main() {
	if err := runtime.Run("search", runtime.WithAssembly(build)); err != nil {
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

	ranking := app.DefaultRanking()
	if err := config.LoadNamed(cfg.Root, cfg.Env, "ranking", &rankingConfig{Ranking: &ranking}); err != nil {
		return nil, err
	}

	service := app.NewService(searchpg.NewRepo(pool.Pool), ranking, currentYear)
	return &runtime.Assembly{
		GRPCServices: []runtime.GRPCService{searchgrpc.NewServer(service)},
		Closers:      []io.Closer{poolCloser{pool}},
	}, nil
}

type rankingConfig struct {
	Ranking *app.Ranking `koanf:"ranking"`
}

type poolCloser struct{ pool *pg.Pool }

func (closer poolCloser) Close() error {
	_ = closer.pool.Close()
	return nil
}
