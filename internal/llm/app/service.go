package app

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"slices"
	"time"

	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/config"
)

const (
	maxAttemptsPerModel = 2
	retryBaseDelay      = 25 * time.Millisecond
)

type Service struct {
	config config.LLM
	client ChatClient
}

func New(cfg config.LLM, client ChatClient) (*Service, error) {
	if cfg.DefaultProvider == "" {
		return nil, ErrProviderNotConfigured
	}
	if _, ok := cfg.Providers[cfg.DefaultProvider]; !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotConfigured, cfg.DefaultProvider)
	}
	return &Service{config: cfg, client: client}, nil
}

func (service *Service) Complete(ctx context.Context, task string, payload map[string]any) (*Result, error) {
	taskCfg, ok := service.config.Tasks[task]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownTask, task)
	}
	models := service.modelChain(taskCfg)
	if len(models) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrModelNotAllowed, taskCfg.Model)
	}

	messages := messagesFromPayload(payload)
	opts := Options{Temperature: taskCfg.Temperature, MaxTokens: taskCfg.MaxTokens, JSON: taskCfg.JSON}

	var lastErr error
	for _, model := range models {
		chat, err := service.attempt(ctx, model, messages, opts, taskCfg)
		if err != nil {
			lastErr = err
			continue
		}
		if chat.Content == "" {
			lastErr = ErrEmptyResponse
			continue
		}
		return &Result{
			Content:      chat.Content,
			Model:        chat.Model,
			InputTokens:  chat.InputTokens,
			OutputTokens: chat.OutputTokens,
			IsJSON:       taskCfg.JSON,
			Valid:        !taskCfg.JSON || isValidJSON(chat.Content),
		}, nil
	}
	return nil, lastErr
}

func (service *Service) attempt(ctx context.Context, model string, messages []Message, opts Options, taskCfg config.LLMTask) (*ChatResult, error) {
	var lastErr error
	for i := range maxAttemptsPerModel {
		if i > 0 {
			if err := sleepBackoff(ctx, i); err != nil {
				return nil, err
			}
		}
		attemptCtx, cancel := context.WithTimeout(ctx, timeout(taskCfg))
		chat, err := service.client.Complete(attemptCtx, model, messages, opts)
		cancel()
		if err == nil {
			return chat, nil
		}
		lastErr = err
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	return nil, lastErr
}

func (service *Service) modelChain(taskCfg config.LLMTask) []string {
	chain := make([]string, 0, 3)
	for _, model := range []string{taskCfg.Model, taskCfg.FallbackModel, taskCfg.EscalateModel} {
		if model != "" && service.allowed(model) && !slices.Contains(chain, model) {
			chain = append(chain, model)
		}
	}
	return chain
}

func (service *Service) allowed(model string) bool {
	return slices.Contains(service.config.Allowlist, model)
}

func sleepBackoff(ctx context.Context, attempt int) error {
	delay := retryBaseDelay*time.Duration(attempt) + time.Duration(rand.Int63n(int64(retryBaseDelay)))
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func timeout(task config.LLMTask) time.Duration {
	if task.TimeoutS > 0 {
		return time.Duration(task.TimeoutS) * time.Second
	}
	return 60 * time.Second
}

func messagesFromPayload(payload map[string]any) []Message {
	if payload == nil {
		return nil
	}
	if raw, ok := payload["messages"].([]any); ok {
		messages := make([]Message, 0, len(raw))
		for _, item := range raw {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			messages = append(messages, Message{
				Role:    stringField(entry, "role"),
				Content: stringField(entry, "content"),
			})
		}
		return messages
	}
	if prompt, ok := payload["prompt"].(string); ok {
		return []Message{{Role: "user", Content: prompt}}
	}
	return nil
}

func stringField(entry map[string]any, key string) string {
	if value, ok := entry[key].(string); ok {
		return value
	}
	return ""
}

func isValidJSON(content string) bool {
	var target any
	return json.Unmarshal([]byte(content), &target) == nil
}
