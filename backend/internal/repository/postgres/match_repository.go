package postgres

import (
	"database/sql"
	"fmt"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

type MatchRepository struct {
	db *sql.DB
}

func NewMatchRepository(db *sql.DB) *MatchRepository {
	return &MatchRepository{db: db}
}

func (r *MatchRepository) Save(match domain.Match) error {
	_, err := r.db.Exec(`
		INSERT INTO matches (id, vulnerability_id, repository_id, component_purl, matched_component, matched_version, status, resolved_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (id) DO NOTHING
	`, match.ID, match.VulnerabilityID, match.RepositoryID, match.ComponentPURL,
		match.MatchedComponent, match.MatchedVersion, match.Status, match.ResolvedAt)
	if err != nil {
		return fmt.Errorf("failed to save match: %w", err)
	}
	return nil
}

func (r *MatchRepository) GetByRepositoryID(repoID string) ([]domain.Match, error) {
	return r.queryMatches(`
		SELECT id, vulnerability_id, repository_id, component_purl, matched_component, matched_version, status, resolved_at, created_at
		FROM matches WHERE repository_id = $1
	`, repoID)
}

func (r *MatchRepository) GetByStatus(status domain.MatchStatus) ([]domain.Match, error) {
	return r.queryMatches(`
		SELECT id, vulnerability_id, repository_id, component_purl, matched_component, matched_version, status, resolved_at, created_at
		FROM matches WHERE status = $1
	`, string(status))
}

func (r *MatchRepository) GetUnresolvedByRepositoryID(repoID string) ([]domain.Match, error) {
	return r.queryMatches(`
		SELECT id, vulnerability_id, repository_id, component_purl, matched_component, matched_version, status, resolved_at, created_at
		FROM matches WHERE repository_id = $1 AND status != 'RESOLVED'
	`, repoID)
}

func (r *MatchRepository) UpdateStatus(id string, status domain.MatchStatus) error {
	_, err := r.db.Exec(`
		UPDATE matches SET status = $1, resolved_at = CASE WHEN $1 = 'RESOLVED' THEN NOW() ELSE NULL END
		WHERE id = $2
	`, string(status), id)
	if err != nil {
		return fmt.Errorf("failed to update match status: %w", err)
	}
	return nil
}

func (r *MatchRepository) queryMatches(query string, args ...any) ([]domain.Match, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query matches: %w", err)
	}
	defer rows.Close()

	var matches []domain.Match
	for rows.Next() {
		var m domain.Match
		err := rows.Scan(&m.ID, &m.VulnerabilityID, &m.RepositoryID, &m.ComponentPURL,
			&m.MatchedComponent, &m.MatchedVersion, &m.Status, &m.ResolvedAt, &m.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan match: %w", err)
		}
		matches = append(matches, m)
	}
	return matches, nil
}
