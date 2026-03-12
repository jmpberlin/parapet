package usecase

import (
	"time"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

type VulnerabilityRepository interface {
	Save(vulnerability domain.Vulnerability) error
	GetAll() ([]domain.Vulnerability, error)
	GetByCVE(cve string) (*domain.Vulnerability, error)
	GetByID(id string) (*domain.Vulnerability, error)
	GetNewerThan(timestamp time.Time) ([]domain.Vulnerability, error)
}

type ArticleRepository interface {
	Save(article domain.Article) error
	GetByID(id string) (*domain.Article, error)
	GetByURL(url string) (*domain.Article, error)
	GetUnprocessed() ([]domain.Article, error)
	MarkProcessed(id string) error
}

type MatchRepository interface {
	GetByRepositoryID(id string) ([]domain.Match, error)
	GetByStatus(status domain.MatchStatus) ([]domain.Match, error)
	GetUnresolvedByRepositoryID(repoID string) ([]domain.Match, error)
	Save(match domain.Match) error
	UpdateStatus(id string, status domain.MatchStatus) error
}

type DependencyRepository interface {
	SaveAll(technologies []domain.RepositoryDependency) error
	GetByRepoID(id string) ([]domain.RepositoryDependency, error)
	DeleteAllByRepoID(id string) error
	DeleteByIDs(ids []string) error
	Save(dep domain.RepositoryDependency) error
	UpdateLastMatchedAt(id string, matchedAt time.Time) error
}

type WatchedRepoRepository interface {
	Save(repository domain.WatchedRepository) error
	Archive(id string) error
	GetByID(id string) (*domain.WatchedRepository, error)
	GetAll() ([]domain.WatchedRepository, error)
	UpdateLastFetchedAt(repoID string, fetchedAt time.Time) error
}

type CrawlerOrchestrator interface {
	FetchArticles(since time.Time) ([]domain.Article, []error)
}

type VulnerabilityExtractor interface {
	ExtractVulnerabilities(articles []domain.Article) []domain.ArticleExtractionResult
}
type DependencyFetcher interface {
	GetDependencies(owner, repo, token string) ([]domain.RepositoryDependency, error)
}
