package app

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/config"
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
	if !service.allowed(taskCfg.Model) {
		return nil, fmt.Errorf("%w: %s", ErrModelNotAllowed, taskCfg.Model)
	}

	ctx, cancel := context.WithTimeout(ctx, timeout(taskCfg))
	defer cancel()

	chat, err := service.client.Complete(ctx, taskCfg.Model, messagesFromPayload(payload), Options{
		Temperature: taskCfg.Temperature,
		MaxTokens:   taskCfg.MaxTokens,
		JSON:        taskCfg.JSON,
	})
	if err != nil {
		return nil, err
	}
	if chat.Content == "" {
		return nil, ErrEmptyResponse
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

func (service *Service) allowed(model string) bool {
	if len(service.config.Allowlist) == 0 {
		return true
	}
	return slices.Contains(service.config.Allowlist, model)
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
