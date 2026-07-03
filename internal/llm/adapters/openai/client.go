package openai

import (
	"context"
	"fmt"
	"strings"

	"github.com/maksim-mshp/nornickel-hackathon/internal/llm/app"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
)

type Client struct {
	client openai.Client
}

func New(baseURL string, apiKey string, authScheme string) *Client {
	options := []option.RequestOption{option.WithHeader("Authorization", authorization(authScheme, apiKey))}
	if baseURL != "" {
		options = append(options, option.WithBaseURL(baseURL))
	}
	return &Client{client: openai.NewClient(options...)}
}

func authorization(scheme string, apiKey string) string {
	if strings.EqualFold(scheme, "api-key") {
		return "Api-Key " + apiKey
	}
	return "Bearer " + apiKey
}

func (client *Client) Complete(ctx context.Context, model string, messages []app.Message, opts app.Options) (*app.ChatResult, error) {
	params := openai.ChatCompletionNewParams{
		Model:    model,
		Messages: toMessages(messages),
	}
	if opts.Temperature != 0 {
		params.Temperature = openai.Float(opts.Temperature)
	}
	if opts.MaxTokens > 0 {
		params.MaxTokens = openai.Int(int64(opts.MaxTokens))
	}
	if opts.JSON {
		params.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &shared.ResponseFormatJSONObjectParam{},
		}
	}

	completion, err := client.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("call upstream: %w", err)
	}

	content := ""
	if len(completion.Choices) > 0 {
		content = completion.Choices[0].Message.Content
	}
	return &app.ChatResult{
		Content:      content,
		Model:        completion.Model,
		InputTokens:  int(completion.Usage.PromptTokens),
		OutputTokens: int(completion.Usage.CompletionTokens),
	}, nil
}

func toMessages(messages []app.Message) []openai.ChatCompletionMessageParamUnion {
	result := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, message := range messages {
		switch message.Role {
		case "system":
			result = append(result, openai.SystemMessage(message.Content))
		case "assistant":
			result = append(result, openai.AssistantMessage(message.Content))
		default:
			result = append(result, openai.UserMessage(message.Content))
		}
	}
	return result
}
