package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
	"github.com/jmpberlin/nightwatch/backend/internal/usecase"
)

type mockArticleRepo struct {
	unprocessed        []domain.Article
	getUnprocessedErr  error
	markProcessedCalls []string
	markProcessedErr   error
}

func (m *mockArticleRepo) GetUnprocessed() ([]domain.Article, error) {
	return m.unprocessed, m.getUnprocessedErr
}

func (m *mockArticleRepo) MarkProcessed(id string) error {
	m.markProcessedCalls = append(m.markProcessedCalls, id)
	return m.markProcessedErr
}

func (m *mockArticleRepo) Save(article domain.Article) error            { return nil }
func (m *mockArticleRepo) GetByID(id string) (*domain.Article, error)   { return nil, nil }
func (m *mockArticleRepo) GetByURL(url string) (*domain.Article, error) { return nil, nil }
func (m *mockArticleRepo) GetByDays(days int) ([]domain.Article, error) { return nil, nil }

type mockVulnerabilityRepo struct {
	saveCalls []domain.Vulnerability
	saveErr   error
}

func (m *mockVulnerabilityRepo) Save(v domain.Vulnerability) error {
	m.saveCalls = append(m.saveCalls, v)
	return m.saveErr
}

func (m *mockVulnerabilityRepo) GetAll() ([]domain.Vulnerability, error)            { return nil, nil }
func (m *mockVulnerabilityRepo) GetByCVE(cve string) (*domain.Vulnerability, error) { return nil, nil }
func (m *mockVulnerabilityRepo) GetByID(id string) (*domain.Vulnerability, error)   { return nil, nil }
func (m *mockVulnerabilityRepo) GetNewerThan(t time.Time) ([]domain.Vulnerability, error) {
	return nil, nil
}

type mockExtractor struct {
	results []domain.ArticleExtractionResult
}

func (m *mockExtractor) ExtractVulnerabilities(articles []domain.Article) []domain.ArticleExtractionResult {
	return m.results
}

func newUseCase(articleRepo *mockArticleRepo, vulnRepo *mockVulnerabilityRepo, extractor *mockExtractor) *usecase.ExtractVulnerabilitiesUseCase {
	return usecase.NewExtractVulnerabilitiesUseCase(vulnRepo, articleRepo, extractor)
}

func TestVulnerabilityExtraction_GetUnprocessedFails_ReturnsEarlyWithError(t *testing.T) {
	articleRepo := &mockArticleRepo{
		getUnprocessedErr: errors.New("db connection failed"),
	}
	vulnRepo := &mockVulnerabilityRepo{}
	extractor := &mockExtractor{}

	result := newUseCase(articleRepo, vulnRepo, extractor).Execute()

	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}
	if len(vulnRepo.saveCalls) != 0 {
		t.Errorf("expected no save calls, got %d", len(vulnRepo.saveCalls))
	}
	if len(articleRepo.markProcessedCalls) != 0 {
		t.Errorf("expected no mark processed calls, got %d", len(articleRepo.markProcessedCalls))
	}
}

func TestVulnerabilityExtraction_NoUnprocessedArticles_ReturnsEarly(t *testing.T) {
	articleRepo := &mockArticleRepo{unprocessed: []domain.Article{}}
	vulnRepo := &mockVulnerabilityRepo{}
	extractor := &mockExtractor{}

	result := newUseCase(articleRepo, vulnRepo, extractor).Execute()

	if len(result.Errors) != 0 {
		t.Errorf("expected no errors, got %d", len(result.Errors))
	}
	if len(vulnRepo.saveCalls) != 0 {
		t.Errorf("expected no save calls, got %d", len(vulnRepo.saveCalls))
	}
}

func TestVulnerabilityExtraction_ExtractionErrorOnOneArticle_OtherArticlesProcessed(t *testing.T) {
	articles := []domain.Article{
		{ID: "article-1", SourceURL: "https://example.com/1"},
		{ID: "article-2", SourceURL: "https://example.com/2"},
	}
	articleRepo := &mockArticleRepo{unprocessed: articles}
	vulnRepo := &mockVulnerabilityRepo{}
	extractor := &mockExtractor{
		results: []domain.ArticleExtractionResult{
			{
				ArticleID: "article-1",
				Err:       errors.New("claude api timeout"),
			},
			{
				ArticleID: "article-2",
				Vulnerabilities: []domain.Vulnerability{
					{CVE: "CVE-2024-1234", Severity: domain.SeverityHigh},
				},
			},
		},
	}

	result := newUseCase(articleRepo, vulnRepo, extractor).Execute()

	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}
	if len(vulnRepo.saveCalls) != 1 {
		t.Errorf("expected 1 vulnerability saved, got %d", len(vulnRepo.saveCalls))
	}
	if vulnRepo.saveCalls[0].CVE != "CVE-2024-1234" {
		t.Errorf("expected CVE-2024-1234 to be saved, got %s", vulnRepo.saveCalls[0].CVE)
	}
	if len(articleRepo.markProcessedCalls) != 1 {
		t.Errorf("expected 1 article marked processed, got %d", len(articleRepo.markProcessedCalls))
	}
	if articleRepo.markProcessedCalls[0] != "article-2" {
		t.Errorf("expected article-2 to be marked processed, got %s", articleRepo.markProcessedCalls[0])
	}
}

func TestVulnerabilityExtraction_NoVulnerabilitiesFound_ArticleMarkedProcessed(t *testing.T) {
	articles := []domain.Article{
		{ID: "article-1", SourceURL: "https://example.com/1"},
	}
	articleRepo := &mockArticleRepo{unprocessed: articles}
	vulnRepo := &mockVulnerabilityRepo{}
	extractor := &mockExtractor{
		results: []domain.ArticleExtractionResult{
			{
				ArticleID:       "article-1",
				Vulnerabilities: []domain.Vulnerability{},
			},
		},
	}

	result := newUseCase(articleRepo, vulnRepo, extractor).Execute()

	if len(result.Errors) != 0 {
		t.Errorf("expected no errors, got %d", len(result.Errors))
	}
	if len(vulnRepo.saveCalls) != 0 {
		t.Errorf("expected no save calls, got %d", len(vulnRepo.saveCalls))
	}
	if len(articleRepo.markProcessedCalls) != 1 {
		t.Errorf("expected article marked processed, got %d", len(articleRepo.markProcessedCalls))
	}
	if articleRepo.markProcessedCalls[0] != "article-1" {
		t.Errorf("expected article-1 marked processed, got %s", articleRepo.markProcessedCalls[0])
	}
}

func TestVulnerabilityExtraction_VulnerabilitySaveFails_ArticleNotMarkedProcessed(t *testing.T) {
	articles := []domain.Article{
		{ID: "article-1", SourceURL: "https://example.com/1"},
	}
	articleRepo := &mockArticleRepo{unprocessed: articles}
	vulnRepo := &mockVulnerabilityRepo{saveErr: errors.New("db write failed")}
	extractor := &mockExtractor{
		results: []domain.ArticleExtractionResult{
			{
				ArticleID: "article-1",
				Vulnerabilities: []domain.Vulnerability{
					{CVE: "CVE-2024-1234", Severity: domain.SeverityHigh},
				},
			},
		},
	}

	result := newUseCase(articleRepo, vulnRepo, extractor).Execute()

	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}
	if len(articleRepo.markProcessedCalls) != 0 {
		t.Errorf("expected article NOT marked processed, got %d calls", len(articleRepo.markProcessedCalls))
	}
}

func TestVulnerabilityExtraction_AllVulnerabilitiesSaved_CountCorrect(t *testing.T) {
	articles := []domain.Article{
		{ID: "article-1", SourceURL: "https://example.com/1"},
		{ID: "article-2", SourceURL: "https://example.com/2"},
	}
	articleRepo := &mockArticleRepo{unprocessed: articles}
	vulnRepo := &mockVulnerabilityRepo{}
	extractor := &mockExtractor{
		results: []domain.ArticleExtractionResult{
			{
				ArticleID: "article-1",
				Vulnerabilities: []domain.Vulnerability{
					{CVE: "CVE-2024-0000", Severity: domain.SeverityHigh},
					{CVE: "CVE-2024-1111", Severity: domain.SeverityCritical},
				},
			},
			{
				ArticleID: "article-2",
				Vulnerabilities: []domain.Vulnerability{
					{CVE: "CVE-2025-0000", Severity: domain.SeverityHigh},
					{CVE: "CVE-2025-1111", Severity: domain.SeverityCritical},
				},
			},
		},
	}

	result := newUseCase(articleRepo, vulnRepo, extractor).Execute()

	if result.VulnerabilitiesExtracted != 4 {
		t.Errorf("expected 4 vulnerabilities extracted, got %d", result.VulnerabilitiesExtracted)
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected no errors, got %d", len(result.Errors))
	}
	if len(articleRepo.markProcessedCalls) != 2 {
		t.Errorf("expected 2 articles marked processed, got %d", len(articleRepo.markProcessedCalls))
	}
}

func TestVulnerabilityExtraction_MarkProcessedFails_ErrorCollectedCountCorrect(t *testing.T) {
	articles := []domain.Article{
		{ID: "article-1", SourceURL: "https://example.com/1"},
	}
	articleRepo := &mockArticleRepo{
		unprocessed:      articles,
		markProcessedErr: errors.New("db write failed"),
	}
	vulnRepo := &mockVulnerabilityRepo{}
	extractor := &mockExtractor{
		results: []domain.ArticleExtractionResult{
			{
				ArticleID: "article-1",
				Vulnerabilities: []domain.Vulnerability{
					{CVE: "CVE-2024-1234", Severity: domain.SeverityHigh},
				},
			},
		},
	}

	result := newUseCase(articleRepo, vulnRepo, extractor).Execute()

	if result.VulnerabilitiesExtracted != 1 {
		t.Errorf("expected 1 vulnerability extracted, got %d", result.VulnerabilitiesExtracted)
	}
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}
}
