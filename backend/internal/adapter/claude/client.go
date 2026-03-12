package claude

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

type claudeClient struct {
	client *anthropic.Client
}

type vulnerabilityDTO struct {
	CVE                  string `json:"CVE"`
	Severity             string `json:"Severity"`
	Description          string `json:"Description"`
	AffectedTechnologies []struct {
		Name         string `json:"Name"`
		VersionRange string `json:"VersionRange"`
		PURL         string `json:"PURL"`
	} `json:"AffectedTechnologies"`
}

var extractionTool = anthropic.ToolParam{
	Name: "extract_vulnerabilities",
	Description: anthropic.String(`Extract software vulnerabilities from a security 
	article, returning only vulnerabilities that affect software developers can 
	install and run in their own infrastructure.`),
	InputSchema: anthropic.ToolInputSchemaParam{
		Type: "object",
		Properties: map[string]any{
			"vulnerabilities": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"CVE": map[string]any{"type": "string"},
						"Severity": map[string]any{
							"type": "string",
							"enum": []string{
								string(domain.SeverityCritical),
								string(domain.SeverityHigh),
								string(domain.SeverityMedium),
								string(domain.SeverityLow),
							},
						},
						"Description": map[string]any{"type": "string"},
						"AffectedTechnologies": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"Name":         map[string]any{"type": "string"},
									"VersionRange": map[string]any{"type": "string"},
									"PURL":         map[string]any{"type": "string"},
								},
							},
						},
					},
				},
			},
		},
	},
}

func NewClaudeClient(apiKey string) *claudeClient {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &claudeClient{client: &client}
}

func (c *claudeClient) ExtractVulnerabilities(articles []domain.Article) []domain.ArticleExtractionResult {
	return processArticles(context.TODO(), c.client, articles)
}

func processArticles(ctx context.Context, client *anthropic.Client, articles []domain.Article) []domain.ArticleExtractionResult {
	results := make(chan domain.ArticleExtractionResult, len(articles))
	semaphore := make(chan struct{}, 5)

	var wg sync.WaitGroup

	for i, a := range articles {
		wg.Add(1)
		go func(idx int, art domain.Article) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			results <- extractFromArticle(ctx, client, art)
		}(i, a)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	return collectResults(results)
}

func extractFromArticle(ctx context.Context, client *anthropic.Client, art domain.Article) domain.ArticleExtractionResult {
	message, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5,
		MaxTokens: 4096,
		System: []anthropic.TextBlockParam{
			{Text: prompt},
		},
		Tools:      []anthropic.ToolUnionParam{{OfTool: &extractionTool}},
		ToolChoice: anthropic.ToolChoiceParamOfTool("extract_vulnerabilities"),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(art.ContentCleaned)),
		},
	})
	if err != nil {
		return domain.ArticleExtractionResult{ArticleID: art.ID, Err: err}
	}

	vulns, err := parseMessageVulnerabilities(message, art)
	if err != nil {
		return domain.ArticleExtractionResult{ArticleID: art.ID, Err: err}
	}

	return domain.ArticleExtractionResult{ArticleID: art.ID, Vulnerabilities: vulns}

}

func parseMessageVulnerabilities(message *anthropic.Message, art domain.Article) ([]domain.Vulnerability, error) {
	var vulns []domain.Vulnerability

	for _, block := range message.Content {
		if block.Type != "tool_use" {
			continue
		}

		var payload struct {
			Vulnerabilities []vulnerabilityDTO `json:"vulnerabilities"`
		}
		if err := json.Unmarshal(block.Input, &payload); err != nil {
			return nil, err
		}

		for _, dto := range payload.Vulnerabilities {
			vulns = append(vulns, toVulnerability(dto, art))
		}
	}

	return vulns, nil
}

func toVulnerability(dto vulnerabilityDTO, art domain.Article) domain.Vulnerability {
	techs := make([]domain.AffectedTechnology, len(dto.AffectedTechnologies))
	for i, t := range dto.AffectedTechnologies {
		techs[i] = domain.AffectedTechnology{Name: t.Name, VersionRange: t.VersionRange, PURL: t.PURL}
	}
	return domain.Vulnerability{
		CVE:                  dto.CVE,
		Severity:             domain.Severity(dto.Severity),
		Description:          dto.Description,
		AffectedTechnologies: techs,
		SourceArticleIDs:     []string{art.ID},
		CreatedAt:            time.Now(),
	}
}

func collectResults(results <-chan domain.ArticleExtractionResult) []domain.ArticleExtractionResult {
	var formatted = []domain.ArticleExtractionResult{}
	for r := range results {
		formatted = append(formatted, r)
	}
	return formatted
}
