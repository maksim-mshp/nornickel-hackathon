package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	answergrpc "github.com/maksim-mshp/nornickel-hackathon/internal/answer/ports/grpc"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/config"
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

func build(cfg config.Bundle, _ *slog.Logger) (*runtime.Assembly, error) {
	target := cfg.Runtime.GRPCClients["search"]
	if target == "" {
		return nil, errors.New("grpc_clients.search is required")
	}

	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create search grpc client: %w", err)
	}

	server := answergrpc.NewServer(kmapv1.NewSearchServiceClient(conn))
	return &runtime.Assembly{
		GRPCServices: []runtime.GRPCService{server},
		Closers:      []io.Closer{conn},
	}, nil
}
