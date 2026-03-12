package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

type WatchedRepoRepository struct {
	db *sql.DB
}

func NewWatchedRepoRepository(db *sql.DB) *WatchedRepoRepository {
	return &WatchedRepoRepository{db: db}
}

func (r *WatchedRepoRepository) Save(repo domain.WatchedRepository) error {
	_, err := r.db.Exec(`
		INSERT INTO watched_repositories (id, git_provider, owner_name, repository_name, integrated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (id) DO NOTHING
	`, repo.ID, string(repo.GitProvider), repo.OwnerName, repo.RepositoryName)
	if err != nil {
		return fmt.Errorf("failed to save watched repository: %w", err)
	}
	return nil
}

func (r *WatchedRepoRepository) Archive(id string) error {
	_, err := r.db.Exec(`
		UPDATE watched_repositories SET archived_at = NOW() WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("failed to archive watched repository: %w", err)
	}
	return nil
}

func (r *WatchedRepoRepository) GetByID(id string) (*domain.WatchedRepository, error) {
	row := r.db.QueryRow(`
		SELECT id, git_provider, owner_name, repository_name, integrated_at, archived_at, last_fetched_at
		FROM watched_repositories WHERE id = $1
	`, id)
	return r.scanWatchedRepo(row)
}

func (r *WatchedRepoRepository) GetAll() ([]domain.WatchedRepository, error) {
	rows, err := r.db.Query(`
		SELECT id, git_provider, owner_name, repository_name, integrated_at, archived_at, last_fetched_at
		FROM watched_repositories
		WHERE archived_at IS NULL
		ORDER BY integrated_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get watched repositories: %w", err)
	}
	defer rows.Close()

	var repos []domain.WatchedRepository
	for rows.Next() {
		var repo domain.WatchedRepository
		var gitProvider string
		err := rows.Scan(
			&repo.ID, &gitProvider, &repo.OwnerName, &repo.RepositoryName,
			&repo.IntegratedAt, &repo.ArchivedAt, &repo.LastFetchedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch from watched repository: %w", err)
		}
		repo.GitProvider = domain.GitProvider(gitProvider)
		repos = append(repos, repo)
	}
	return repos, nil
}

func (r *WatchedRepoRepository) UpdateLastFetchedAt(repoID string, fetchedAt time.Time) error {
	_, err := r.db.Exec(`
		UPDATE watched_repositories SET last_fetched_at = $1 WHERE id = $2
	`, fetchedAt, repoID)
	if err != nil {
		return fmt.Errorf("failed to update last_fetched_at: %w", err)
	}
	return nil
}

func (r *WatchedRepoRepository) scanWatchedRepo(row *sql.Row) (*domain.WatchedRepository, error) {
	var repo domain.WatchedRepository
	var gitProvider string
	err := row.Scan(
		&repo.ID, &gitProvider, &repo.OwnerName, &repo.RepositoryName,
		&repo.IntegratedAt, &repo.ArchivedAt, &repo.LastFetchedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan watched repository: %w", err)
	}
	repo.GitProvider = domain.GitProvider(gitProvider)
	return &repo, nil
}
