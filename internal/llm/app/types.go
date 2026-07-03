package app

import (
	"context"
	"errors"
)

var (
	ErrUnknownTask          = errors.New("unknown llm task")
	ErrModelNotAllowed      = errors.New("model is not in allowlist")
	ErrProviderNotConfigured = errors.New("llm provider not configured")
	ErrEmptyResponse        = errors.New("empty completion response")
)

type Message struct {
	Role    string
	Content string
}

type Options struct {
	Temperature     float64
	MaxTokens       int
	JSON            bool
	ReasoningEffort string
}

type ChatResult struct {
	Content      string
	Model        string
	InputTokens  int
	OutputTokens int
}

type ChatClient interface {
	Complete(ctx context.Context, model string, messages []Message, opts Options) (*ChatResult, error)
}

type Result struct {
	Content      string
	Model        string
	InputTokens  int
	OutputTokens int
	Valid        bool
	IsJSON       bool
}
