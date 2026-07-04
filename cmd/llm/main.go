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

func buildAssembly(cfg config.Bundle, logger *slog.Logger) (*runtime.Assembly, error) {
	llmCfg := cfg.Runtime.LLM
	if _, ok := llmCfg.Providers[llmCfg.DefaultProvider]; !ok {
		return nil, fmt.Errorf("%w: %s", app.ErrProviderNotConfigured, llmCfg.DefaultProvider)
	}

	var providers []openai.Provider
	for _, name := range append([]string{llmCfg.DefaultProvider}, llmCfg.FallbackProviders...) {
		provider, ok := llmCfg.Providers[name]
		if !ok {
			return nil, fmt.Errorf("%w: %s", app.ErrProviderNotConfigured, name)
		}
		if provider.APIKey == "" || provider.BaseURL == "" {
			logger.Warn("llm provider skipped: base_url/api_key missing", "provider", name)
			continue
		}
		providers = append(providers, openai.Provider{
			Name:   name,
			Client: openai.New(provider.BaseURL, provider.APIKey, provider.AuthScheme, provider.FolderID),
			Models: provider.Models,
		})
	}
	if len(providers) == 0 {
		return nil, errors.New("llm: no provider has both base_url and api_key")
	}

	client, err := openai.NewFallback(logger, providers...)
	if err != nil {
		return nil, err
	}

	service, err := app.New(llmCfg, client)
	if err != nil {
		return nil, err
	}

	return &runtime.Assembly{
		GRPCServices: []runtime.GRPCService{llmgrpc.NewServer(service)},
	}, nil
}
