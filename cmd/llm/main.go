package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/maksim-mshp/nornickel-hackathon/internal/llm/adapters/openai"
	"github.com/maksim-mshp/nornickel-hackathon/internal/llm/app"
	llmgrpc "github.com/maksim-mshp/nornickel-hackathon/internal/llm/ports/grpc"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/config"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/runtime"
)

func main() {
	if err := runtime.Run("llm", runtime.WithAssembly(buildAssembly)); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func buildAssembly(cfg config.Bundle, _ *slog.Logger) (*runtime.Assembly, error) {
	llmCfg := cfg.Runtime.LLM
	provider, ok := llmCfg.Providers[llmCfg.DefaultProvider]
	if !ok {
		return nil, fmt.Errorf("%w: %s", app.ErrProviderNotConfigured, llmCfg.DefaultProvider)
	}
	if provider.APIKey == "" || provider.BaseURL == "" {
		return nil, errors.New("llm provider base_url and api_key are required")
	}

	service, err := app.New(llmCfg, openai.New(provider.BaseURL, provider.APIKey, provider.AuthScheme))
	if err != nil {
		return nil, err
	}

	return &runtime.Assembly{
		GRPCServices: []runtime.GRPCService{llmgrpc.NewServer(service)},
	}, nil
}
