package postgres_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmpberlin/nightwatch/backend/internal/domain"
	postgres "github.com/jmpberlin/nightwatch/backend/internal/repository/postgres"
)

func truncateMatches(t *testing.T) {
	t.Helper()
	_, err := testDB.Exec(`TRUNCATE TABLE matches`)
	if err != nil {
		t.Fatalf("failed to truncate matches: %s", err)
	}
}

func seedMatchPrereqs(t *testing.T) (vulnID, repoID string) {
	t.Helper()
	vulnID = uuid.New().String()
	repoID = uuid.New().String()

	_, err := testDB.Exec(`
		INSERT INTO vulnerabilities (id, severity, description, source_article_ids, published_at)
		VALUES ($1, 'HIGH', 'test vulnerability', '{}', NOW())
	`, vulnID)
	if err != nil {
		t.Fatalf("failed to seed vulnerability: %s", err)
	}

	_, err = testDB.Exec(`
		INSERT INTO watched_repositories (id, git_provider, owner_name, repository_name)
		VALUES ($1, 'github', 'testowner', $2)
	`, repoID, uuid.New().String())
	if err != nil {
		t.Fatalf("failed to seed watched repository: %s", err)
	}

	return
}

func newTestMatch(vulnID, repoID string) domain.Match {
	return domain.Match{
		ID:               uuid.New().String(),
		VulnerabilityID:  vulnID,
		RepositoryID:     repoID,
		ComponentPURL:    "pkg:npm/lodash@4.17.20",
		MatchedComponent: "lodash",
		MatchedVersion:   "4.17.20",
		Status:           domain.MatchStatusWarning,
	}
}

func findMatchByID(matches []domain.Match, id string) *domain.Match {
	for i := range matches {
		if matches[i].ID == id {
			return &matches[i]
		}
	}
	return nil
}

func TestMatchRepository_Save_AndRetrieve(t *testing.T) {
	truncateMatches(t)
	vulnID, repoID := seedMatchPrereqs(t)
	repo := postgres.NewMatchRepository(testDB)

	m := newTestMatch(vulnID, repoID)
	if err := repo.Save(m); err != nil {
		t.Fatalf("failed to save match: %s", err)
	}

	matches, err := repo.GetByRepositoryID(repoID)
	if err != nil {
		t.Fatalf("failed to get matches: %s", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}

	got := matches[0]
	if got.ID != m.ID {
		t.Errorf("expected ID %s, got %s", m.ID, got.ID)
	}
	if got.VulnerabilityID != m.VulnerabilityID {
		t.Errorf("expected VulnerabilityID %s, got %s", m.VulnerabilityID, got.VulnerabilityID)
	}
	if got.ComponentPURL != m.ComponentPURL {
		t.Errorf("expected ComponentPURL %s, got %s", m.ComponentPURL, got.ComponentPURL)
	}
	if got.Status != domain.MatchStatusWarning {
		t.Errorf("expected status WARNING, got %s", got.Status)
	}
}

func TestMatchRepository_Save_EvidenceFields(t *testing.T) {
	truncateMatches(t)
	vulnID, repoID := seedMatchPrereqs(t)
	repo := postgres.NewMatchRepository(testDB)

	m := newTestMatch(vulnID, repoID)
	m.Confidence = "HIGH"
	m.MatchedOn = "purl-exact"
	m.VulnIdentifier = "pkg:npm/lodash@4.17.20"
	m.DepIdentifier = "pkg:npm/lodash@4.17.20"

	if err := repo.Save(m); err != nil {
		t.Fatalf("failed to save match: %s", err)
	}

	matches, err := repo.GetByRepositoryID(repoID)
	if err != nil {
		t.Fatalf("failed to get matches: %s", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}

	got := matches[0]
	if got.Confidence != "HIGH" {
		t.Errorf("expected Confidence HIGH, got %q", got.Confidence)
	}
	if got.MatchedOn != "purl-exact" {
		t.Errorf("expected MatchedOn purl-exact, got %q", got.MatchedOn)
	}
	if got.VulnIdentifier != "pkg:npm/lodash@4.17.20" {
		t.Errorf("expected VulnIdentifier pkg:npm/lodash@4.17.20, got %q", got.VulnIdentifier)
	}
	if got.DepIdentifier != "pkg:npm/lodash@4.17.20" {
		t.Errorf("expected DepIdentifier pkg:npm/lodash@4.17.20, got %q", got.DepIdentifier)
	}
}

func TestMatchRepository_GetByRepositoryID_IsolatedByRepo(t *testing.T) {
	truncateMatches(t)
	vulnID1, repoID1 := seedMatchPrereqs(t)
	vulnID2, repoID2 := seedMatchPrereqs(t)
	repo := postgres.NewMatchRepository(testDB)

	if err := repo.Save(newTestMatch(vulnID1, repoID1)); err != nil {
		t.Fatalf("failed to save match 1: %s", err)
	}
	m2 := newTestMatch(vulnID2, repoID2)
	if err := repo.Save(m2); err != nil {
		t.Fatalf("failed to save match 2: %s", err)
	}

	matches, err := repo.GetByRepositoryID(repoID1)
	if err != nil {
		t.Fatalf("failed to get matches: %s", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match for repo1, got %d", len(matches))
	}
	if matches[0].RepositoryID != repoID1 {
		t.Errorf("expected match belonging to repo1, got repo %s", matches[0].RepositoryID)
	}
}

func TestMatchRepository_GetByStatus(t *testing.T) {
	truncateMatches(t)
	vulnID, repoID := seedMatchPrereqs(t)
	repo := postgres.NewMatchRepository(testDB)

	warning := newTestMatch(vulnID, repoID)

	confirmed := newTestMatch(vulnID, repoID)
	confirmed.ComponentPURL = "pkg:npm/express@4.18.0"
	confirmed.MatchedComponent = "express"
	confirmed.Status = domain.MatchStatusConfirmed

	if err := repo.Save(warning); err != nil {
		t.Fatalf("failed to save warning match: %s", err)
	}
	if err := repo.Save(confirmed); err != nil {
		t.Fatalf("failed to save confirmed match: %s", err)
	}

	warnings, err := repo.GetByStatus(domain.MatchStatusWarning)
	if err != nil {
		t.Fatalf("failed to get WARNING matches: %s", err)
	}
	if len(warnings) != 1 || warnings[0].ID != warning.ID {
		t.Errorf("expected 1 WARNING match, got %d", len(warnings))
	}

	confirmeds, err := repo.GetByStatus(domain.MatchStatusConfirmed)
	if err != nil {
		t.Fatalf("failed to get CONFIRMED matches: %s", err)
	}
	if len(confirmeds) != 1 || confirmeds[0].ID != confirmed.ID {
		t.Errorf("expected 1 CONFIRMED match, got %d", len(confirmeds))
	}
}

func TestMatchRepository_GetUnresolvedByRepositoryID(t *testing.T) {
	truncateMatches(t)
	vulnID, repoID := seedMatchPrereqs(t)
	repo := postgres.NewMatchRepository(testDB)

	active := newTestMatch(vulnID, repoID)

	resolved := newTestMatch(vulnID, repoID)
	resolved.ComponentPURL = "pkg:npm/express@4.18.0"
	resolved.MatchedComponent = "express"
	resolved.Status = domain.MatchStatusResolved

	if err := repo.Save(active); err != nil {
		t.Fatalf("failed to save active match: %s", err)
	}
	if err := repo.Save(resolved); err != nil {
		t.Fatalf("failed to save resolved match: %s", err)
	}

	unresolved, err := repo.GetUnresolvedByRepositoryID(repoID)
	if err != nil {
		t.Fatalf("failed to get unresolved matches: %s", err)
	}
	if len(unresolved) != 1 {
		t.Fatalf("expected 1 unresolved match, got %d", len(unresolved))
	}
	if unresolved[0].ID != active.ID {
		t.Errorf("expected active match in result, got %s", unresolved[0].ID)
	}
}

func TestMatchRepository_Save_DuplicateID_Ignored(t *testing.T) {
	truncateMatches(t)
	vulnID, repoID := seedMatchPrereqs(t)
	repo := postgres.NewMatchRepository(testDB)

	m := newTestMatch(vulnID, repoID)
	if err := repo.Save(m); err != nil {
		t.Fatalf("failed to save original match: %s", err)
	}

	duplicate := m
	duplicate.MatchedVersion = "9.9.9"
	if err := repo.Save(duplicate); err != nil {
		t.Fatalf("second save with same ID should not return an error: %s", err)
	}

	matches, err := repo.GetByRepositoryID(repoID)
	if err != nil {
		t.Fatalf("failed to get matches: %s", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match after duplicate save, got %d", len(matches))
	}
	if matches[0].MatchedVersion != m.MatchedVersion {
		t.Errorf("expected original version %s to be kept, got %s", m.MatchedVersion, matches[0].MatchedVersion)
	}
}

func TestMatchRepository_Save_DuplicateVulnAndPURL_Ignored(t *testing.T) {
	truncateMatches(t)
	vulnID, repoID := seedMatchPrereqs(t)
	repo := postgres.NewMatchRepository(testDB)

	first := newTestMatch(vulnID, repoID)
	first.Status = domain.MatchStatusWarning
	if err := repo.Save(first); err != nil {
		t.Fatalf("failed to save first match: %s", err)
	}

	second := newTestMatch(vulnID, repoID)
	second.ID = uuid.New().String()
	second.Status = domain.MatchStatusConfirmed
	if err := repo.Save(second); err != nil {
		t.Fatalf("second save with same vuln+repo+purl should not return an error: %s", err)
	}

	matches, err := repo.GetByRepositoryID(repoID)
	if err != nil {
		t.Fatalf("failed to get matches: %s", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match after duplicate vuln+purl save, got %d", len(matches))
	}
	if matches[0].ID != first.ID {
		t.Errorf("expected first match to be kept, got ID %s", matches[0].ID)
	}
	if matches[0].Status != domain.MatchStatusWarning {
		t.Errorf("expected original status WARNING to be kept, got %s", matches[0].Status)
	}
}

func TestMatchRepository_Save_DuplicateVulnAndComponentNoPURL_Ignored(t *testing.T) {
	truncateMatches(t)
	vulnID, repoID := seedMatchPrereqs(t)
	repo := postgres.NewMatchRepository(testDB)

	first := newTestMatch(vulnID, repoID)
	first.ComponentPURL = ""
	first.Status = domain.MatchStatusWarning
	if err := repo.Save(first); err != nil {
		t.Fatalf("failed to save first match: %s", err)
	}

	second := newTestMatch(vulnID, repoID)
	second.ID = uuid.New().String()
	second.ComponentPURL = ""
	second.Status = domain.MatchStatusConfirmed
	if err := repo.Save(second); err != nil {
		t.Fatalf("second save with same vuln+repo+component (no purl) should not return an error: %s", err)
	}

	matches, err := repo.GetByRepositoryID(repoID)
	if err != nil {
		t.Fatalf("failed to get matches: %s", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match after duplicate vuln+component save, got %d", len(matches))
	}
	if matches[0].ID != first.ID {
		t.Errorf("expected first match to be kept, got ID %s", matches[0].ID)
	}
	if matches[0].Status != domain.MatchStatusWarning {
		t.Errorf("expected original status WARNING to be kept, got %s", matches[0].Status)
	}
}

func TestMatchRepository_UpdateStatus_ToResolved(t *testing.T) {
	truncateMatches(t)
	vulnID, repoID := seedMatchPrereqs(t)
	repo := postgres.NewMatchRepository(testDB)

	m := newTestMatch(vulnID, repoID)
	if err := repo.Save(m); err != nil {
		t.Fatalf("failed to save match: %s", err)
	}

	before := time.Now().Add(-time.Second)
	if err := repo.UpdateStatus(m.ID, domain.MatchStatusResolved); err != nil {
		t.Fatalf("failed to update status: %s", err)
	}

	matches, err := repo.GetByRepositoryID(repoID)
	if err != nil {
		t.Fatalf("failed to get matches: %s", err)
	}
	got := findMatchByID(matches, m.ID)
	if got == nil {
		t.Fatal("match not found after update")
	}
	if got.Status != domain.MatchStatusResolved {
		t.Errorf("expected status RESOLVED, got %s", got.Status)
	}
	if got.ResolvedAt == nil {
		t.Fatal("expected ResolvedAt to be set after resolving")
	}
	if got.ResolvedAt.Before(before) {
		t.Errorf("expected ResolvedAt to be recent, got %s", got.ResolvedAt)
	}
}

func TestMatchRepository_UpdateStatus_ClearsResolvedAt(t *testing.T) {
	truncateMatches(t)
	vulnID, repoID := seedMatchPrereqs(t)
	repo := postgres.NewMatchRepository(testDB)

	m := newTestMatch(vulnID, repoID)
	if err := repo.Save(m); err != nil {
		t.Fatalf("failed to save match: %s", err)
	}

	if err := repo.UpdateStatus(m.ID, domain.MatchStatusResolved); err != nil {
		t.Fatalf("failed to update to RESOLVED: %s", err)
	}
	if err := repo.UpdateStatus(m.ID, domain.MatchStatusWarning); err != nil {
		t.Fatalf("failed to update back to WARNING: %s", err)
	}

	matches, err := repo.GetByRepositoryID(repoID)
	if err != nil {
		t.Fatalf("failed to get matches: %s", err)
	}
	got := findMatchByID(matches, m.ID)
	if got == nil {
		t.Fatal("match not found after update")
	}
	if got.Status != domain.MatchStatusWarning {
		t.Errorf("expected status WARNING, got %s", got.Status)
	}
	if got.ResolvedAt != nil {
		t.Errorf("expected ResolvedAt to be nil after re-opening, got %s", got.ResolvedAt)
	}
}
