package openai

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/maksim-mshp/nornickel-hackathon/internal/llm/app"
)

type Provider struct {
	Name   string
	Client app.ChatClient
	Models map[string]string
}

func (p Provider) resolve(model string) (string, bool) {
	if len(p.Models) == 0 {
		return model, true
	}
	mapped, ok := p.Models[model]
	return mapped, ok
}

type FallbackClient struct {
	providers []Provider
	log       *slog.Logger
}

func NewFallback(log *slog.Logger, providers ...Provider) (*FallbackClient, error) {
	if len(providers) == 0 {
		return nil, errors.New("llm: no providers configured")
	}
	if log == nil {
		log = slog.Default()
	}
	return &FallbackClient{providers: providers, log: log}, nil
}

func (f *FallbackClient) Complete(ctx context.Context, model string, messages []app.Message, opts app.Options) (*app.ChatResult, error) {
	var lastErr error
	for index, provider := range f.providers {
		providerModel, ok := provider.resolve(model)
		if !ok {
			continue
		}
		result, err := provider.Client.Complete(ctx, providerModel, messages, opts)
		if err == nil {
			if index > 0 {
				f.log.Info("llm fallback provider served request",
					"provider", provider.Name, "model", providerModel, "canonical", model)
			}
			return result, nil
		}
		lastErr = err
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		f.log.Warn("llm provider failed, trying next",
			"provider", provider.Name, "model", providerModel, "canonical", model, "err", err)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("%w: %s", app.ErrModelNotAllowed, model)
	}
	return nil, lastErr
}
