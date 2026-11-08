package postgres

import (
	"database/sql"
	"fmt"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

// matchColumns is the column list shared by every match query.
const matchColumns = "id, vulnerability_id, repository_id, component_purl, matched_component, matched_version, status, resolved_at, created_at, confidence, matched_on, vuln_identifier, dep_identifier, version_check"

type MatchRepository struct {
	db *sql.DB
}

func NewMatchRepository(db *sql.DB) *MatchRepository {
	return &MatchRepository{db: db}
}

func (r *MatchRepository) Save(match domain.Match) error {
	_, err := r.db.Exec(`
		INSERT INTO matches (id, vulnerability_id, repository_id, component_purl, matched_component, matched_version, status, resolved_at, created_at, confidence, matched_on, vuln_identifier, dep_identifier, version_check)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), $9, $10, $11, $12, $13)
		ON CONFLICT DO NOTHING
	`, match.ID, match.VulnerabilityID, match.RepositoryID, match.ComponentPURL,
		match.MatchedComponent, match.MatchedVersion, match.Status, match.ResolvedAt,
		match.Confidence, match.MatchedOn, match.VulnIdentifier, match.DepIdentifier, match.VersionCheck)
	if err != nil {
		return fmt.Errorf("failed to save match: %w", err)
	}
	return nil
}

func (r *MatchRepository) GetByRepositoryID(repoID string) ([]domain.Match, error) {
	return r.queryMatches(`
		SELECT `+matchColumns+`
		FROM matches WHERE repository_id = $1
	`, repoID)
}

func (r *MatchRepository) GetByRepositoryIDPaginated(repoID string, page, limit int) ([]domain.Match, int, error) {
	total, err := r.GetCountByRepositoryID(repoID)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	matches, err := r.queryMatches(`
		SELECT `+matchColumns+`
		FROM matches WHERE repository_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, repoID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return matches, total, nil
}

func (r *MatchRepository) GetByID(id string) (*domain.Match, error) {
	matches, err := r.queryMatches(`
			SELECT `+matchColumns+`
			FROM matches WHERE id = $1
	`, id)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, nil
	}
	return &matches[0], nil
}

func (r *MatchRepository) GetCountByRepositoryID(repoID string) (int, error) {
	var total int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM matches WHERE repository_id = $1
	`, repoID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to count matches: %w", err)
	}
	return total, nil
}

func (r *MatchRepository) GetByStatus(status domain.MatchStatus) ([]domain.Match, error) {
	return r.queryMatches(`
		SELECT `+matchColumns+`
		FROM matches WHERE status = $1
	`, string(status))
}

func (r *MatchRepository) GetUnresolvedByRepositoryID(repoID string) ([]domain.Match, error) {
	return r.queryMatches(`
		SELECT `+matchColumns+`
		FROM matches WHERE repository_id = $1 AND status != 'RESOLVED'
	`, repoID)
}

func (r *MatchRepository) UpdateStatus(id string, status domain.MatchStatus) error {
	_, err := r.db.Exec(`
		UPDATE matches SET status = $1, resolved_at = CASE WHEN $2 = 'RESOLVED' THEN NOW() ELSE NULL END
		WHERE id = $3
	`, string(status), string(status), id)
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
		var confidence, matchedOn, vulnIdentifier, depIdentifier, versionCheck sql.NullString
		err := rows.Scan(&m.ID, &m.VulnerabilityID, &m.RepositoryID, &m.ComponentPURL,
			&m.MatchedComponent, &m.MatchedVersion, &m.Status, &m.ResolvedAt, &m.CreatedAt,
			&confidence, &matchedOn, &vulnIdentifier, &depIdentifier, &versionCheck)
		if err != nil {
			return nil, fmt.Errorf("failed to scan match: %w", err)
		}
		m.Confidence = confidence.String
		m.MatchedOn = matchedOn.String
		m.VulnIdentifier = vulnIdentifier.String
		m.DepIdentifier = depIdentifier.String
		m.VersionCheck = versionCheck.String
		matches = append(matches, m)
	}
	return matches, nil
}
