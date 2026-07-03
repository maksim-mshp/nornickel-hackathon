package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/maksim-mshp/nornickel-hackathon/internal/llm/app"
)

const (
	maxResponseBytes = 8 << 20
	errorSnippet     = 512
)

type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func New(baseURL string, apiKey string) *Client {
	return &Client{baseURL: baseURL, apiKey: apiKey, http: &http.Client{}}
}

type request struct {
	Model          string          `json:"model"`
	Messages       []message       `json:"messages"`
	Temperature    float64         `json:"temperature,omitempty"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type response struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

type errorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

func (client *Client) Complete(ctx context.Context, model string, messages []app.Message, opts app.Options) (*app.ChatResult, error) {
	body, err := json.Marshal(buildRequest(model, messages, opts))
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+client.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := client.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call upstream: %w", err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(httpResp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		var errResp errorResponse
		_ = json.Unmarshal(raw, &errResp)
		detail := snippet(raw)
		if errResp.Error.Message != "" {
			detail = errResp.Error.Message
		}
		return nil, fmt.Errorf("upstream status %d: %s", httpResp.StatusCode, detail)
	}

	var parsed response
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	content := ""
	if len(parsed.Choices) > 0 {
		content = parsed.Choices[0].Message.Content
	}
	return &app.ChatResult{
		Content:      content,
		Model:        parsed.Model,
		InputTokens:  parsed.Usage.PromptTokens,
		OutputTokens: parsed.Usage.CompletionTokens,
	}, nil
}

func buildRequest(model string, messages []app.Message, opts app.Options) request {
	req := request{
		Model:       model,
		Messages:    make([]message, 0, len(messages)),
		Temperature: opts.Temperature,
		MaxTokens:   opts.MaxTokens,
	}
	for _, msg := range messages {
		req.Messages = append(req.Messages, message{Role: msg.Role, Content: msg.Content})
	}
	if opts.JSON {
		req.ResponseFormat = &responseFormat{Type: "json_object"}
	}
	return req
}

func snippet(raw []byte) string {
	if len(raw) > errorSnippet {
		return string(raw[:errorSnippet])
	}
	return string(raw)
}
