package usecase

import "github.com/jmpberlin/nightwatch/backend/internal/domain"

type VulnerabilityRepository interface {
	Save(vulnerability domain.Vulnerability) error
	GetAll() ([]domain.Vulnerability, error)
	GetByCVE(cve string) (*domain.Vulnerability, error)
	GetByID(id string) (*domain.Vulnerability, error)
	GetUnprocessed() ([]domain.Vulnerability, error)
}

type ArticleRepository interface {
	Save(article domain.Article) error
	GetByID(id string) (*domain.Article, error)
	GetByURL(url string) (*domain.Article, error)
	GetUnprocessed() ([]domain.Article, error)
}

type MatchRepository interface {
	GetByRepositoryID(id string) ([]domain.Match, error)
	GetByStatus(status domain.MatchStatus) ([]domain.Match, error)
	GetUnresolvedByRepositoryID(repoID string) ([]domain.Match, error)
	Save(match domain.Match) error
	UpdateStatus(id string, status domain.MatchStatus) error
}

type RepositoryDependencyRepository interface {
	SaveAll(technologies []domain.RepositoryDependency) error
	GetByRepoID(id string) ([]domain.RepositoryDependency, error)
	DeleteByRepoID(id string) error
}

type WatchedRepoRepository interface {
	Save(repository domain.WatchedRepository) error
	Archive(id string) error
	GetByID(id string) (*domain.WatchedRepository, error)
	GetAll() ([]domain.WatchedRepository, error)
}
