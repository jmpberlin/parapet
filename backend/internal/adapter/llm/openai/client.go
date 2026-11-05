package openai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	oai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	"github.com/jmpberlin/nightwatch/backend/internal/adapter/llm/shared"
	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

const (
	maxTokensIncludingReasoning = 16000
	requestTimeoutPerAttempt    = 120 * time.Second
	maxRetries                  = 1
)

type Config struct {
	BaseURL string
	APIKey  string
	Model   string
}

type Client struct {
	client oai.Client
	model  string
}

func NewClient(cfg Config) (*Client, error) {
	if cfg.BaseURL == "" || cfg.APIKey == "" || cfg.Model == "" {
		return nil, errors.New("openai: base URL, API key and model are required")
	}
	return &Client{
		client: oai.NewClient(
			option.WithBaseURL(cfg.BaseURL),
			option.WithAPIKey(cfg.APIKey),
			option.WithMaxRetries(maxRetries),
			option.WithRequestTimeout(requestTimeoutPerAttempt),
		),
		model: cfg.Model,
	}, nil
}

func (c *Client) ExtractVulnerabilities(articles []domain.Article) []domain.ArticleExtractionResult {
	ctx := context.Background()
	return shared.ProcessConcurrently(articles, func(art domain.Article) domain.ArticleExtractionResult {
		vulns, err := c.extract(ctx, art)
		return domain.ArticleExtractionResult{ArticleID: art.ID, Vulnerabilities: vulns, Err: err}
	})
}

func (c *Client) extract(ctx context.Context, art domain.Article) ([]domain.Vulnerability, error) {
	resp, err := c.client.Chat.Completions.New(ctx, oai.ChatCompletionNewParams{
		Model:     c.model,
		MaxTokens: oai.Int(maxTokensIncludingReasoning),
		Messages: []oai.ChatCompletionMessageParamUnion{
			oai.SystemMessage(shared.Prompt),
			oai.UserMessage(art.ContentCleaned),
		},
		Tools: []oai.ChatCompletionToolUnionParam{
			oai.ChatCompletionFunctionTool(oai.FunctionDefinitionParam{
				Name:        shared.ToolName,
				Description: oai.String(shared.ToolDescription),
				Parameters:  oai.FunctionParameters(shared.Schema()),
			}),
		},
		ToolChoice: oai.ToolChoiceOptionFunctionToolChoice(oai.ChatCompletionNamedToolChoiceFunctionParam{
			Name: shared.ToolName,
		}),
	})
	var apiErr *oai.Error
	if errors.As(err, &apiErr) {
		return nil, fmt.Errorf("chat completions: HTTP %d: %s", apiErr.StatusCode, apiErr.Message)
	}
	if err != nil {
		return nil, fmt.Errorf("chat completions: %w", err)
	}
	return parseResponse(resp, art)
}

func parseResponse(resp *oai.ChatCompletion, art domain.Article) ([]domain.Vulnerability, error) {
	if len(resp.Choices) == 0 {
		return nil, errors.New("chat completions: response has no choices")
	}
	choice := resp.Choices[0]

	if args, ok := toolCallArguments(choice.Message.ToolCalls); ok {
		vulns, err := shared.ParsePayload([]byte(args), art)
		if err != nil {
			return nil, fmt.Errorf("tool call arguments: %w", err)
		}
		return vulns, nil
	}

	content := stripCodeFence(choice.Message.Content)
	if content == "" {
		return nil, fmt.Errorf("chat completions: no tool call and empty content (finish_reason=%q)", choice.FinishReason)
	}
	vulns, err := shared.ParsePayloadStrict([]byte(content), art)
	if err != nil {
		return nil, fmt.Errorf("no tool call; content fallback: %w", err)
	}
	return vulns, nil
}

func toolCallArguments(calls []oai.ChatCompletionMessageToolCallUnion) (string, bool) {
	for _, call := range calls {
		if call.Function.Name == shared.ToolName {
			return call.Function.Arguments, true
		}
	}
	if len(calls) > 0 {
		return calls[0].Function.Arguments, true
	}
	return "", false
}

func stripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	s = strings.TrimPrefix(s, "```")
	if languageTagEnd := strings.IndexByte(s, '\n'); languageTagEnd >= 0 {
		s = s[languageTagEnd+1:]
	}
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "```"))
}
