package pg

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	DSN      string
	MaxConns int32
}

type Pool struct {
	*pgxpool.Pool
}

func New(ctx context.Context, cfg Config) (*Pool, error) {
	if cfg.DSN == "" {
		return nil, errors.New("postgres dsn is required")
	}

	parsed, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse postgres dsn: %w", err)
	}
	if cfg.MaxConns > 0 {
		parsed.MaxConns = cfg.MaxConns
	}

	pool, err := pgxpool.NewWithConfig(ctx, parsed)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	return &Pool{Pool: pool}, nil
}
