package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	answerpg "github.com/maksim-mshp/nornickel-hackathon/internal/answer/adapters/pg"
	answerapp "github.com/maksim-mshp/nornickel-hackathon/internal/answer/app"
	answerconsumer "github.com/maksim-mshp/nornickel-hackathon/internal/answer/ports/consumer"
	answergrpc "github.com/maksim-mshp/nornickel-hackathon/internal/answer/ports/grpc"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/auth"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/config"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/nats"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/pg"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if err := runtime.Run("answer", runtime.WithAssembly(build)); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func build(cfg config.Bundle, logger *slog.Logger) (*runtime.Assembly, error) {
	target := cfg.Runtime.GRPCClients["search"]
	if target == "" {
		return nil, errors.New("grpc_clients.search is required")
	}

	signingKey := []byte(cfg.Runtime.Auth.SigningKey)
	conn, err := grpc.NewClient(target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(auth.UnaryClientInterceptor(signingKey)),
		grpc.WithChainStreamInterceptor(auth.StreamClientInterceptor(signingKey)),
	)
	if err != nil {
		return nil, fmt.Errorf("create search grpc client: %w", err)
	}

	pool, err := pg.New(context.Background(), pg.Config{
		DSN:      cfg.Runtime.Postgres.DSN,
		MaxConns: cfg.Runtime.Postgres.MaxConns,
	})
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	cache := answerpg.NewCache(pool.Pool, time.Duration(cfg.Runtime.Cache.TTLHours)*time.Hour)

	closers := []io.Closer{conn, pool}
	options := []answerapp.Option{answerapp.WithCache(cache)}
	if llmTarget := cfg.Runtime.GRPCClients["llm"]; llmTarget != "" {
		llmConn, llmErr := grpc.NewClient(llmTarget,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithChainUnaryInterceptor(auth.UnaryClientInterceptor(signingKey)),
		)
		if llmErr != nil {
			_ = conn.Close()
			_ = pool.Close()
			return nil, fmt.Errorf("create llm grpc client: %w", llmErr)
		}
		options = append(options, answerapp.WithSynthesizer(answerapp.NewLLMSynthesizer(kmapv1.NewLLMServiceClient(llmConn))))
		closers = append(closers, llmConn)
	}
	server := answergrpc.NewServer(kmapv1.NewSearchServiceClient(conn), options...)

	var workers []runtime.Worker
	if url := cfg.Runtime.NATS.URL; url != "" {
		bus, busErr := nats.New(context.Background(), nats.Config{
			URL:     url,
			Name:    "kmap-answer",
			Streams: []nats.StreamSpec{{Name: "KMAP_FACTS", Subjects: []string{"kmap.facts.v1.>"}}},
		})
		if busErr != nil {
			logger.Warn("answer cache invalidation disabled", "error", busErr)
		} else {
			workers = append(workers, answerconsumer.NewWorker(bus, cache, logger))
			closers = append(closers, bus)
		}
	}

	return &runtime.Assembly{
		GRPCServices: []runtime.GRPCService{server},
		Workers:      workers,
		Closers:      closers,
	}, nil
}
