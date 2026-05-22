package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

type mockWatchedRepoRepo struct{}

func (m *mockWatchedRepoRepo) Save(r domain.WatchedRepository) error                { return nil }
func (m *mockWatchedRepoRepo) Archive(id string) error                              { return nil }
func (m *mockWatchedRepoRepo) GetByID(id string) (*domain.WatchedRepository, error) { return nil, nil }
func (m *mockWatchedRepoRepo) GetAll() ([]domain.WatchedRepository, error)          { return nil, nil }
func (m *mockWatchedRepoRepo) UpdateLastFetchedAt(id string, t time.Time) error     { return nil }

type mockDependencyRepo struct{}

func (m *mockDependencyRepo) Save(d domain.RepositoryDependency) error      { return nil }
func (m *mockDependencyRepo) SaveAll(d []domain.RepositoryDependency) error { return nil }
func (m *mockDependencyRepo) GetByRepoID(id string) ([]domain.RepositoryDependency, error) {
	return nil, nil
}
func (m *mockDependencyRepo) GetByRepoIDOrderedByLastMatchedAt(id string) ([]domain.RepositoryDependency, error) {
	return nil, nil
}
func (m *mockDependencyRepo) DeleteAllByRepoID(id string) error                { return nil }
func (m *mockDependencyRepo) DeleteByIDs(ids []string) error                   { return nil }
func (m *mockDependencyRepo) UpdateLastMatchedAt(id string, t time.Time) error { return nil }

type mockMatchRepo struct{}

func (m *mockMatchRepo) Save(match domain.Match) error                            { return nil }
func (m *mockMatchRepo) GetByRepositoryID(id string) ([]domain.Match, error)      { return nil, nil }
func (m *mockMatchRepo) GetByStatus(s domain.MatchStatus) ([]domain.Match, error) { return nil, nil }
func (m *mockMatchRepo) GetUnresolvedByRepositoryID(id string) ([]domain.Match, error) {
	return nil, nil
}
func (m *mockMatchRepo) UpdateStatus(id string, s domain.MatchStatus) error { return nil }

type mockVulnerabilityRepo struct {
	vulns []domain.Vulnerability
	err   error
}

func (m *mockVulnerabilityRepo) GetNewerThan(t time.Time) ([]domain.Vulnerability, error) {
	return m.vulns, m.err
}
func (m *mockVulnerabilityRepo) Save(v domain.Vulnerability) error                  { return nil }
func (m *mockVulnerabilityRepo) GetAll() ([]domain.Vulnerability, error)            { return nil, nil }
func (m *mockVulnerabilityRepo) GetByCVE(cve string) (*domain.Vulnerability, error) { return nil, nil }
func (m *mockVulnerabilityRepo) GetByID(id string) (*domain.Vulnerability, error)   { return nil, nil }

func TestFindMatch_PURLMatch_ReturnsConfirmedMatch(t *testing.T) {
	vuln := domain.Vulnerability{
		ID: "vuln-1",
		AffectedTechnologies: []domain.AffectedTechnology{
			{Name: "react", PURL: "pkg:npm/react@18.0.0", VersionRange: "> 1.0.0"},
		},
	}
	dep := domain.RepositoryDependency{
		ID:      "dep-1",
		Name:    "react",
		PURL:    "pkg:npm/react@17.0.0",
		Version: "17.0.0",
	}

	result, ok := findMatch(vuln, dep, "repo-1")

	if !ok {
		t.Fatal("expected a match, got none")
	}
	if result.Status != domain.MatchStatusConfirmed {
		t.Errorf("expected CONFIRMED, got %s", result.Status)
	}
	if result.VulnerabilityID != "vuln-1" {
		t.Errorf("expected vuln-1, got %s", result.VulnerabilityID)
	}
	if result.ComponentPURL != "pkg:npm/react@17.0.0" {
		t.Errorf("expected dep PURL, got %s", result.ComponentPURL)
	}
}

func TestFindMatch_PURLMatch_NoVersionRange_ReturnsWarning(t *testing.T) {
	vuln := domain.Vulnerability{
		ID: "vuln-1",
		AffectedTechnologies: []domain.AffectedTechnology{
			{Name: "react", PURL: "pkg:npm/react@18.0.0", VersionRange: ""},
		},
	}
	dep := domain.RepositoryDependency{
		ID:      "dep-1",
		Name:    "react",
		PURL:    "pkg:npm/react@17.0.0",
		Version: "17.0.0",
	}

	result, ok := findMatch(vuln, dep, "repo-1")

	if !ok {
		t.Fatal("expected a match, got none")
	}
	if result.Status != domain.MatchStatusWarning {
		t.Errorf("expected WARNING, got %s", result.Status)
	}
}

func TestFindMatch_PURLMatch_NoDepVersion_ReturnsWarning(t *testing.T) {
	vuln := domain.Vulnerability{
		ID: "vuln-1",
		AffectedTechnologies: []domain.AffectedTechnology{
			{Name: "react", PURL: "pkg:npm/react@18.0.0", VersionRange: "> 1.0.0"},
		},
	}
	dep := domain.RepositoryDependency{
		ID:      "dep-1",
		Name:    "react",
		PURL:    "pkg:npm/react",
		Version: "",
	}

	result, ok := findMatch(vuln, dep, "repo-1")

	if !ok {
		t.Fatal("expected a match, got none")
	}
	if result.Status != domain.MatchStatusWarning {
		t.Errorf("expected WARNING, got %s", result.Status)
	}
}

func TestFindMatch_DifferentPURL_NoMatch(t *testing.T) {
	vuln := domain.Vulnerability{
		ID: "vuln-1",
		AffectedTechnologies: []domain.AffectedTechnology{
			{Name: "react", PURL: "pkg:npm/react@18.0.0"},
		},
	}
	dep := domain.RepositoryDependency{
		ID:   "dep-1",
		Name: "lodash",
		PURL: "pkg:npm/lodash@4.0.0",
	}

	_, ok := findMatch(vuln, dep, "repo-1")

	if ok {
		t.Error("expected no match — different PURLs should not match")
	}
}

func TestFindMatch_NameFallback_BothPURLsEmpty_ReturnsMatch(t *testing.T) {
	vuln := domain.Vulnerability{
		ID: "vuln-1",
		AffectedTechnologies: []domain.AffectedTechnology{
			{Name: "express", PURL: ""},
		},
	}
	dep := domain.RepositoryDependency{
		ID:   "dep-1",
		Name: "express",
		PURL: "",
	}

	_, ok := findMatch(vuln, dep, "repo-1")

	if !ok {
		t.Error("expected name fallback match when both PURLs are empty")
	}
}

func TestFindMatch_NoAffectedTechnologies_NoMatch(t *testing.T) {
	vuln := domain.Vulnerability{
		ID:                   "vuln-1",
		AffectedTechnologies: []domain.AffectedTechnology{},
	}
	dep := domain.RepositoryDependency{
		ID:   "dep-1",
		Name: "react",
		PURL: "pkg:npm/react@18.0.0",
	}

	_, ok := findMatch(vuln, dep, "repo-1")

	if ok {
		t.Error("expected no match when vulnerability has no affected technologies")
	}
}

func newMatchUseCase(vulnRepo VulnerabilityRepository) *MatchVulnerabilitiesUseCase {
	return NewMatchVulnerabilitiesUseCase(
		&mockWatchedRepoRepo{},
		&mockDependencyRepo{},
		vulnRepo,
		&mockMatchRepo{},
	)
}

func TestPrepareMatchingData_AllUnmatched_FetchesAllVulns(t *testing.T) {
	vulns := []domain.Vulnerability{{ID: "vuln-1"}}
	vulnRepo := &mockVulnerabilityRepo{vulns: vulns}
	uc := newMatchUseCase(vulnRepo)

	deps := []domain.RepositoryDependency{
		{ID: "dep-1", LastMatchedAt: nil},
		{ID: "dep-2", LastMatchedAt: nil},
	}

	unmatched, matched, allVulns, latestVulns, err := uc.prepareMatchingData(deps)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(unmatched) != 2 {
		t.Errorf("expected 2 unmatched deps, got %d", len(unmatched))
	}
	if len(matched) != 0 {
		t.Errorf("expected 0 matched deps, got %d", len(matched))
	}
	if len(allVulns) != 1 {
		t.Errorf("expected 1 vuln in allVulns, got %d", len(allVulns))
	}
	if len(latestVulns) != 0 {
		t.Errorf("expected 0 latestVulns, got %d", len(latestVulns))
	}
}

func TestPrepareMatchingData_AllMatched_FetchesLatestVulnsOnly(t *testing.T) {
	vulns := []domain.Vulnerability{{ID: "vuln-1"}}
	vulnRepo := &mockVulnerabilityRepo{vulns: vulns}
	uc := newMatchUseCase(vulnRepo)

	now := time.Now()
	deps := []domain.RepositoryDependency{
		{ID: "dep-1", LastMatchedAt: &now},
		{ID: "dep-2", LastMatchedAt: &now},
	}

	unmatched, matched, allVulns, latestVulns, err := uc.prepareMatchingData(deps)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(unmatched) != 0 {
		t.Errorf("expected 0 unmatched deps, got %d", len(unmatched))
	}
	if len(matched) != 2 {
		t.Errorf("expected 2 matched deps, got %d", len(matched))
	}
	if len(allVulns) != 0 {
		t.Errorf("expected 0 allVulns, got %d", len(allVulns))
	}
	if len(latestVulns) != 1 {
		t.Errorf("expected 1 latestVuln, got %d", len(latestVulns))
	}
}

func TestPrepareMatchingData_Mixed_FetchesBoth(t *testing.T) {
	vulns := []domain.Vulnerability{{ID: "vuln-1"}}
	vulnRepo := &mockVulnerabilityRepo{vulns: vulns}
	uc := newMatchUseCase(vulnRepo)

	now := time.Now()
	deps := []domain.RepositoryDependency{
		{ID: "dep-1", LastMatchedAt: nil},
		{ID: "dep-2", LastMatchedAt: &now},
	}

	unmatched, matched, allVulns, latestVulns, err := uc.prepareMatchingData(deps)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(unmatched) != 1 {
		t.Errorf("expected 1 unmatched dep, got %d", len(unmatched))
	}
	if len(matched) != 1 {
		t.Errorf("expected 1 matched dep, got %d", len(matched))
	}
	if len(allVulns) != 1 {
		t.Errorf("expected 1 allVuln, got %d", len(allVulns))
	}
	if len(latestVulns) != 1 {
		t.Errorf("expected 1 latestVuln, got %d", len(latestVulns))
	}
}

func TestPrepareMatchingData_VulnFetchFails_ReturnsError(t *testing.T) {
	vulnRepo := &mockVulnerabilityRepo{err: errors.New("db failed")}
	uc := newMatchUseCase(vulnRepo)

	deps := []domain.RepositoryDependency{
		{ID: "dep-1", LastMatchedAt: nil},
	}

	_, _, _, _, err := uc.prepareMatchingData(deps)

	if err == nil {
		t.Error("expected error, got nil")
	}
}
