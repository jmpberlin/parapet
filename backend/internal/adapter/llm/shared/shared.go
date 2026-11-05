package shared

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

const (
	ToolName        = "extract_vulnerabilities"
	ToolDescription = `Extract software vulnerabilities from a security
	article, returning only vulnerabilities that affect software developers can
	install and run in their own infrastructure.`
	MaxConcurrency = 5
)

func SchemaProperties() map[string]any {
	return map[string]any{
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
	}
}

func Schema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": SchemaProperties(),
		"required":   []string{"vulnerabilities"},
	}
}

type VulnerabilityDTO struct {
	CVE                  string `json:"CVE"`
	Severity             string `json:"Severity"`
	Description          string `json:"Description"`
	AffectedTechnologies []struct {
		Name         string `json:"Name"`
		VersionRange string `json:"VersionRange"`
		PURL         string `json:"PURL"`
	} `json:"AffectedTechnologies"`
}

type Payload struct {
	Vulnerabilities []VulnerabilityDTO `json:"vulnerabilities"`
}

func ParsePayload(raw []byte, art domain.Article) ([]domain.Vulnerability, error) {
	var payload Payload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("decode extraction payload: %w", err)
	}

	vulns := make([]domain.Vulnerability, 0, len(payload.Vulnerabilities))
	for _, dto := range payload.Vulnerabilities {
		vulns = append(vulns, ToVulnerability(dto, art))
	}
	return vulns, nil
}

func ParsePayloadStrict(raw []byte, art domain.Article) ([]domain.Vulnerability, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, fmt.Errorf("decode extraction payload: %w", err)
	}
	if _, ok := fields["vulnerabilities"]; !ok {
		return nil, fmt.Errorf("decode extraction payload: missing %q key", "vulnerabilities")
	}
	return ParsePayload(raw, art)
}

func ToVulnerability(dto VulnerabilityDTO, art domain.Article) domain.Vulnerability {
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

func ProcessConcurrently(articles []domain.Article, extract func(domain.Article) domain.ArticleExtractionResult) []domain.ArticleExtractionResult {
	results := make(chan domain.ArticleExtractionResult, len(articles))
	semaphore := make(chan struct{}, MaxConcurrency)

	var wg sync.WaitGroup
	for _, a := range articles {
		wg.Add(1)
		go func(art domain.Article) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			results <- extract(art)
		}(a)
	}

	wg.Wait()
	close(results)

	collected := make([]domain.ArticleExtractionResult, 0, len(articles))
	for r := range results {
		collected = append(collected, r)
	}
	return collected
}
