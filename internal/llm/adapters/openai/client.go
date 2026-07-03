package openai

import (
	"context"
	"fmt"
	"strings"

	"github.com/maksim-mshp/nornickel-hackathon/internal/llm/app"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/responses"
	"github.com/openai/openai-go/shared"
)

const jsonInstruction = "Верни ответ строго как валидный JSON без пояснений и текста вокруг."

type Client struct {
	client   openai.Client
	folderID string
}

func New(baseURL string, apiKey string, authScheme string, folderID string) *Client {
	options := []option.RequestOption{option.WithHeader("Authorization", authorization(authScheme, apiKey))}
	if baseURL != "" {
		options = append(options, option.WithBaseURL(baseURL))
	}
	return &Client{client: openai.NewClient(options...), folderID: folderID}
}

func authorization(scheme string, apiKey string) string {
	if strings.EqualFold(scheme, "api-key") {
		return "Api-Key " + apiKey
	}
	return "Bearer " + apiKey
}

func (client *Client) modelURI(model string) string {
	if client.folderID == "" || strings.Contains(model, "://") {
		return model
	}
	return "gpt://" + client.folderID + "/" + model
}

func (client *Client) Complete(ctx context.Context, model string, messages []app.Message, opts app.Options) (*app.ChatResult, error) {
	instructions, input := splitMessages(messages)
	if opts.JSON {
		instructions = strings.TrimSpace(instructions + "\n\n" + jsonInstruction)
	}

	params := responses.ResponseNewParams{
		Model: client.modelURI(model),
		Input: responses.ResponseNewParamsInputUnion{OfString: openai.String(input)},
	}
	if instructions != "" {
		params.Instructions = openai.String(instructions)
	}
	if opts.MaxTokens > 0 {
		params.MaxOutputTokens = openai.Int(int64(opts.MaxTokens))
	}
	if opts.Temperature >= 0 {
		params.Temperature = openai.Float(opts.Temperature)
	}
	if opts.ReasoningEffort != "" {
		params.Reasoning = shared.ReasoningParam{Effort: shared.ReasoningEffort(opts.ReasoningEffort)}
	}

	response, err := client.client.Responses.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("call upstream: %w", err)
	}
	return &app.ChatResult{
		Content:      response.OutputText(),
		Model:        response.Model,
		InputTokens:  int(response.Usage.InputTokens),
		OutputTokens: int(response.Usage.OutputTokens),
	}, nil
}

func splitMessages(messages []app.Message) (instructions string, input string) {
	var systemParts []string
	var inputParts []string
	for _, message := range messages {
		if message.Role == "system" {
			systemParts = append(systemParts, message.Content)
			continue
		}
		inputParts = append(inputParts, message.Content)
	}
	return strings.Join(systemParts, "\n\n"), strings.Join(inputParts, "\n\n")
}
