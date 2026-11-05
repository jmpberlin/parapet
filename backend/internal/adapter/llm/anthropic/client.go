package anthropic

import (
	"context"

	anthropicsdk "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/jmpberlin/nightwatch/backend/internal/adapter/llm/shared"
	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

const maxTokens = 4096

type Config struct {
	BaseURL string
	APIKey  string
	Model   string
}

type Client struct {
	client *anthropicsdk.Client
	model  string
}

var extractionTool = anthropicsdk.ToolParam{
	Name:        shared.ToolName,
	Description: anthropicsdk.String(shared.ToolDescription),
	InputSchema: anthropicsdk.ToolInputSchemaParam{
		Type:       "object",
		Properties: shared.SchemaProperties(),
	},
}

func NewClient(cfg Config) *Client {
	opts := []option.RequestOption{option.WithAPIKey(cfg.APIKey)}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}
	client := anthropicsdk.NewClient(opts...)
	return &Client{client: &client, model: cfg.Model}
}

func (c *Client) ExtractVulnerabilities(articles []domain.Article) []domain.ArticleExtractionResult {
	ctx := context.TODO()
	return shared.ProcessConcurrently(articles, func(art domain.Article) domain.ArticleExtractionResult {
		vulns, err := c.extract(ctx, art)
		return domain.ArticleExtractionResult{ArticleID: art.ID, Vulnerabilities: vulns, Err: err}
	})
}

func (c *Client) extract(ctx context.Context, art domain.Article) ([]domain.Vulnerability, error) {
	message, err := c.client.Messages.New(ctx, anthropicsdk.MessageNewParams{
		Model:      anthropicsdk.Model(c.model),
		MaxTokens:  maxTokens,
		System:     []anthropicsdk.TextBlockParam{{Text: shared.Prompt}},
		Tools:      []anthropicsdk.ToolUnionParam{{OfTool: &extractionTool}},
		ToolChoice: anthropicsdk.ToolChoiceParamOfTool(shared.ToolName),
		Messages: []anthropicsdk.MessageParam{
			anthropicsdk.NewUserMessage(anthropicsdk.NewTextBlock(art.ContentCleaned)),
		},
	})
	if err != nil {
		return nil, err
	}

	var vulns []domain.Vulnerability
	for _, block := range message.Content {
		if block.Type != "tool_use" {
			continue
		}
		parsed, err := shared.ParsePayload(block.Input, art)
		if err != nil {
			return nil, err
		}
		vulns = append(vulns, parsed...)
	}
	return vulns, nil
}
