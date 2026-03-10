package usecase

import "github.com/jmpberlin/nightwatch/backend/internal/domain"

type RepositoryScanner interface {
	GetDependencies(owner, repo, token string) ([]domain.RepositoryDependency, error)
}
