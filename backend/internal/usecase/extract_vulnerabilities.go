package usecase

import "github.com/jmpberlin/nightwatch/backend/internal/domain"

type VulnerabilityExtractor interface {
	ExtractVulnerabilities(article domain.Article) ([]domain.Vulnerability, error)
}
