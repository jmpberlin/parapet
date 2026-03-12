package usecase

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

type UpdateDependenciesStageResult struct {
	ReposUpdated int
	DepsAdded    int
	DepsRemoved  int
	Errors       []error
}

type UpdateDependenciesUseCase struct {
	watchedRepoRepo WatchedRepoRepository
	dependencyRepo  DependencyRepository
	fetcher         DependencyFetcher
	githubToken     string
}

func NewUpdateDependenciesUseCase(watchedRepoRepo WatchedRepoRepository, dependencyRepo DependencyRepository, fetcher DependencyFetcher, githubToken string) *UpdateDependenciesUseCase {
	return &UpdateDependenciesUseCase{
		watchedRepoRepo: watchedRepoRepo,
		dependencyRepo:  dependencyRepo,
		fetcher:         fetcher,
		githubToken:     githubToken,
	}
}

func (u *UpdateDependenciesUseCase) Execute() UpdateDependenciesStageResult {
	result := UpdateDependenciesStageResult{}

	repos, err := u.watchedRepoRepo.GetAll()
	if err != nil {
		slog.Error("failed to fetch watched repositories",
			"stage", StageUpdateDeps,
			"err", err,
		)
		result.Errors = append(result.Errors, fmt.Errorf("%s: fetch watched repos: %w", StageUpdateDeps, err))
		return result
	}

	if len(repos) == 0 {
		slog.Info("no watched repositories found", "stage", StageUpdateDeps)
		return result
	}

	for _, repo := range repos {
		added, removed, errs := u.updateRepo(repo)
		result.DepsAdded += added
		result.DepsRemoved += removed
		result.Errors = append(result.Errors, errs...)
		if len(errs) == 0 {
			result.ReposUpdated++
		}
	}

	slog.Info("update dependencies stage complete",
		"stage", StageUpdateDeps,
		"repos_updated", result.ReposUpdated,
		"deps_added", result.DepsAdded,
		"deps_removed", result.DepsRemoved,
		"errors", len(result.Errors),
	)
	return result
}

func (u *UpdateDependenciesUseCase) updateRepo(repo domain.WatchedRepository) (int, int, []error) {
	githubDeps, storedDeps, err := u.fetchDeps(repo)
	if err != nil {
		return 0, 0, []error{err}
	}

	newDeps, removedIDs := diffDependencies(githubDeps, storedDeps, repo.ID)
	return u.persistAndUpdate(repo, newDeps, removedIDs)
}

func (u *UpdateDependenciesUseCase) fetchDeps(repo domain.WatchedRepository) ([]domain.RepositoryDependency, []domain.RepositoryDependency, error) {
	githubDeps, err := u.fetcher.GetDependencies(repo.OwnerName, repo.RepositoryName, u.githubToken)
	if err != nil {
		slog.Error("failed to fetch dependencies from github",
			"stage", StageUpdateDeps,
			"repo", repo.RepositoryName,
			"err", err,
		)
		return nil, nil, fmt.Errorf("%s: fetch github deps for %s: %w", StageUpdateDeps, repo.RepositoryName, err)
	}

	storedDeps, err := u.dependencyRepo.GetByRepoID(repo.ID)
	if err != nil {
		slog.Error("failed to fetch stored dependencies",
			"stage", StageUpdateDeps,
			"repo", repo.RepositoryName,
			"err", err,
		)
		return nil, nil, fmt.Errorf("%s: fetch stored deps for %s: %w", StageUpdateDeps, repo.RepositoryName, err)
	}

	return githubDeps, storedDeps, nil
}

func (u *UpdateDependenciesUseCase) persistAndUpdate(repo domain.WatchedRepository, newDeps []domain.RepositoryDependency, removedIDs []string) (int, int, []error) {
	added, removed, errs := u.persistDiff(repo, newDeps, removedIDs)
	if len(errs) > 0 {
		slog.Warn("skipping last_fetched_at update — errors during dependency update",
			"stage", StageUpdateDeps,
			"repo", repo.RepositoryName,
		)
		return added, removed, errs
	}

	if err := u.updateLastFetchedAt(repo); err != nil {
		errs = append(errs, err)
	}
	return added, removed, errs
}

func (u *UpdateDependenciesUseCase) persistDiff(repo domain.WatchedRepository, newDeps []domain.RepositoryDependency, removedIDs []string) (int, int, []error) {
	var errs []error
	added := 0
	removed := 0

	for _, dep := range newDeps {
		if err := u.dependencyRepo.Save(dep); err != nil {
			slog.Error("failed to save new dependency",
				"stage", StageUpdateDeps,
				"repo", repo.RepositoryName,
				"dep", dep.Name,
				"err", err,
			)
			errs = append(errs, fmt.Errorf("%s: save dep %s for %s: %w", StageUpdateDeps, dep.Name, repo.RepositoryName, err))
			continue
		}
		added++
	}

	if err := u.dependencyRepo.DeleteByIDs(removedIDs); err != nil {
		slog.Error("failed to delete removed dependencies",
			"stage", StageUpdateDeps,
			"repo", repo.RepositoryName,
			"err", err,
		)
		errs = append(errs, fmt.Errorf("%s: delete deps for %s: %w", StageUpdateDeps, repo.RepositoryName, err))
	} else {
		removed = len(removedIDs)
	}

	return added, removed, errs
}

func (u *UpdateDependenciesUseCase) updateLastFetchedAt(repo domain.WatchedRepository) error {
	if err := u.watchedRepoRepo.UpdateLastFetchedAt(repo.ID, time.Now()); err != nil {
		slog.Error("failed to update last_fetched_at",
			"stage", StageUpdateDeps,
			"repo", repo.RepositoryName,
			"err", err,
		)
		return fmt.Errorf("%s: update last_fetched_at for %s: %w", StageUpdateDeps, repo.RepositoryName, err)
	}
	return nil
}

func diffDependencies(githubDeps []domain.RepositoryDependency, storedDeps []domain.RepositoryDependency, repoID string) (newDeps []domain.RepositoryDependency, removedIDs []string) {
	storedByPURL := make(map[string]domain.RepositoryDependency)
	storedByName := make(map[string]domain.RepositoryDependency)
	for _, d := range storedDeps {
		if d.PURL != "" {
			storedByPURL[d.PURL] = d
		} else {
			storedByName[d.Name] = d
		}
	}

	githubByPURL := make(map[string]domain.RepositoryDependency)
	githubByName := make(map[string]domain.RepositoryDependency)
	for _, d := range githubDeps {
		if d.PURL != "" {
			githubByPURL[d.PURL] = d
		} else {
			githubByName[d.Name] = d
		}
		if d.PURL != "" {
			if _, exists := storedByPURL[d.PURL]; !exists {
				d.ID = domain.NewID()
				d.RepositoryID = repoID
				newDeps = append(newDeps, d)
			}
		} else {
			if _, exists := storedByName[d.Name]; !exists {
				d.ID = domain.NewID()
				d.RepositoryID = repoID
				newDeps = append(newDeps, d)
			}
		}
	}

	for _, d := range storedDeps {
		if d.PURL != "" {
			if _, exists := githubByPURL[d.PURL]; !exists {
				removedIDs = append(removedIDs, d.ID)
			}
		} else {
			if _, exists := githubByName[d.Name]; !exists {
				removedIDs = append(removedIDs, d.ID)
			}
		}
	}

	return newDeps, removedIDs
}
