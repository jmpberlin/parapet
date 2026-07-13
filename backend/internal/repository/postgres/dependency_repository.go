package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
	"github.com/lib/pq"
)

type DependencyRepository struct {
	db *sql.DB
}

func NewDependencyRepository(db *sql.DB) *DependencyRepository {
	return &DependencyRepository{db: db}
}

func (r *DependencyRepository) Save(dep domain.RepositoryDependency) error {
	_, err := r.db.Exec(`
		INSERT INTO repository_dependencies (id, repository_id, name, version, purl, created_at, last_matched_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), $6)
		ON CONFLICT (id) DO NOTHING
	`, dep.ID, dep.RepositoryID, dep.Name, dep.Version, dep.PURL, dep.LastMatchedAt)
	if err != nil {
		return fmt.Errorf("failed to save dependency %s: %w", dep.Name, err)
	}
	return nil
}

func (r *DependencyRepository) SaveAll(deps []domain.RepositoryDependency) error {
	for _, dep := range deps {
		if err := r.Save(dep); err != nil {
			return err
		}
	}
	return nil
}

func (r *DependencyRepository) GetByRepoID(repoID string) ([]domain.RepositoryDependency, error) {
	rows, err := r.db.Query(`
		SELECT id, repository_id, name, version, purl, created_at, last_matched_at
		FROM repository_dependencies
		WHERE repository_id = $1
	`, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependencies for repo %s: %w", repoID, err)
	}
	defer rows.Close()

	var deps []domain.RepositoryDependency
	for rows.Next() {
		var d domain.RepositoryDependency
		err := rows.Scan(&d.ID, &d.RepositoryID, &d.Name, &d.Version, &d.PURL, &d.CreatedAt, &d.LastMatchedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan dependency: %w", err)
		}
		deps = append(deps, d)
	}
	return deps, nil
}

func (r *DependencyRepository) GetByRepoIDPaginated(repoID string, page, limit int) ([]domain.RepositoryDependency, int, error) {
	total, err := r.GetCountByRepoID(repoID)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	rows, err := r.db.Query(`
		SELECT id, repository_id, name, version, purl, created_at, last_matched_at
		FROM repository_dependencies
		WHERE repository_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, repoID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get dependencies for repo %s: %w", repoID, err)
	}
	defer rows.Close()

	var deps []domain.RepositoryDependency
	for rows.Next() {
		var d domain.RepositoryDependency
		err := rows.Scan(&d.ID, &d.RepositoryID, &d.Name, &d.Version, &d.PURL, &d.CreatedAt, &d.LastMatchedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan dependency: %w", err)
		}
		deps = append(deps, d)
	}
	return deps, total, nil
}

func (r *DependencyRepository) GetCountByRepoID(repoID string) (int, error) {
	var total int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM repository_dependencies WHERE repository_id = $1
	`, repoID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to count dependencies: %w", err)
	}
	return total, nil
}

func (r *DependencyRepository) DeleteAllByRepoID(repoID string) error {
	_, err := r.db.Exec(`
		DELETE FROM repository_dependencies WHERE repository_id = $1
	`, repoID)
	if err != nil {
		return fmt.Errorf("failed to delete dependencies for repo %s: %w", repoID, err)
	}
	return nil
}

func (r *DependencyRepository) DeleteByIDs(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.db.Exec(`
		DELETE FROM repository_dependencies WHERE id = ANY($1)
	`, pq.Array(ids))
	if err != nil {
		return fmt.Errorf("failed to delete dependencies by ids: %w", err)
	}
	return nil
}

func (r *DependencyRepository) UpdateLastMatchedAt(id string, matchedAt time.Time) error {
	_, err := r.db.Exec(`
		UPDATE repository_dependencies SET last_matched_at = $1 WHERE id = $2
	`, matchedAt, id)
	if err != nil {
		return fmt.Errorf("failed to update last matched at for dependency %s: %w", id, err)
	}
	return nil
}

func (r *DependencyRepository) GetByRepoIDOrderedByLastMatchedAt(repoID string) ([]domain.RepositoryDependency, error) {
	rows, err := r.db.Query(`
		SELECT id, repository_id, name, version, purl, created_at, last_matched_at
		FROM repository_dependencies
		WHERE repository_id = $1
		ORDER BY last_matched_at ASC NULLS FIRST
	`, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependencies ordered by last_matched_at for repo %s: %w", repoID, err)
	}
	defer rows.Close()

	var deps []domain.RepositoryDependency
	for rows.Next() {
		var d domain.RepositoryDependency
		err := rows.Scan(&d.ID, &d.RepositoryID, &d.Name, &d.Version, &d.PURL, &d.CreatedAt, &d.LastMatchedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan dependency: %w", err)
		}
		deps = append(deps, d)
	}
	return deps, nil
}
