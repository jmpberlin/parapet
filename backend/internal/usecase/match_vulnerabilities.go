package usecase

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
	"github.com/package-url/packageurl-go"
)

const defaultVulnerabilityLookback = 365 * 24 * time.Hour

type MatchStageResult struct {
	MatchesFound int
	Errors       []error
}

type MatchVulnerabilitiesUseCase struct {
	watchedRepoRepo   WatchedRepoRepository
	dependencyRepo    DependencyRepository
	vulnerabilityRepo VulnerabilityRepository
	matchRepository   MatchRepository
}

func NewMatchVulnerabilitiesUseCase(
	watchedRepoRepo WatchedRepoRepository,
	dependencyRepo DependencyRepository,
	vulnerabilityRepo VulnerabilityRepository,
	matchRepository MatchRepository,
) *MatchVulnerabilitiesUseCase {
	return &MatchVulnerabilitiesUseCase{
		watchedRepoRepo:   watchedRepoRepo,
		dependencyRepo:    dependencyRepo,
		vulnerabilityRepo: vulnerabilityRepo,
		matchRepository:   matchRepository,
	}
}

func (m *MatchVulnerabilitiesUseCase) Execute() MatchStageResult {
	result := MatchStageResult{}

	repos, err := m.watchedRepoRepo.GetAll()
	if err != nil {
		slog.Error("failed to fetch watched repositories",
			"stage", StageMatch,
			"err", err,
		)
		result.Errors = append(result.Errors, fmt.Errorf("%s: fetch watched repos: %w", StageMatch, err))
		return result
	}

	if len(repos) == 0 {
		slog.Info("no watched repositories found", "stage", StageMatch)
		return result
	}

	for _, repo := range repos {
		found, errs := m.matchRepo(repo)
		result.MatchesFound += found
		result.Errors = append(result.Errors, errs...)
	}

	slog.Info("match stage complete",
		"stage", StageMatch,
		"matches_found", result.MatchesFound,
		"errors", len(result.Errors),
	)
	return result
}

func (m *MatchVulnerabilitiesUseCase) matchRepo(repo domain.WatchedRepository) (int, []error) {
	deps, err := m.dependencyRepo.GetByRepoIDOrderedByLastMatchedAt(repo.ID)
	if err != nil {
		slog.Error("failed to fetch dependencies",
			"stage", StageMatch,
			"repo", repo.RepositoryName,
			"err", err,
		)
		return 0, []error{fmt.Errorf("%s: fetch deps for %s: %w", StageMatch, repo.RepositoryName, err)}
	}

	if len(deps) == 0 {
		slog.Info("no dependencies found",
			"stage", StageMatch,
			"repo", repo.RepositoryName,
		)
		return 0, nil
	}

	unmatchedDeps, matchedDeps, allVulns, latestVulns, err := m.prepareMatchingData(deps)
	if err != nil {
		slog.Error("failed to prepare matching data",
			"stage", StageMatch,
			"repo", repo.RepositoryName,
			"err", err,
		)
		return 0, []error{fmt.Errorf("%s: prepare matching data for %s: %w", StageMatch, repo.RepositoryName, err)}
	}

	if len(allVulns) == 0 && len(latestVulns) == 0 {
		slog.Info("no vulnerabilities to match against",
			"stage", StageMatch,
			"repo", repo.RepositoryName,
		)
		return 0, m.updateLastMatched(deps)
	}

	allMatches := findAllMatches(unmatchedDeps, matchedDeps, allVulns, latestVulns, repo.ID)

	found, errs := m.persistMatches(repo, allMatches)
	if len(errs) > 0 {
		return found, errs
	}

	return found, m.updateLastMatched(deps)
}

func (m *MatchVulnerabilitiesUseCase) prepareMatchingData(deps []domain.RepositoryDependency) (
	unmatchedDeps []domain.RepositoryDependency,
	matchedDeps []domain.RepositoryDependency,
	allVulns []domain.Vulnerability,
	latestVulns []domain.Vulnerability,
	err error,
) {
	unmatchedDeps, matchedDeps = splitByMatchedStatus(deps)

	if len(unmatchedDeps) > 0 {
		allVulns, err = m.fetchVulnerabilitiesSince(time.Now().Add(-defaultVulnerabilityLookback))
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("fetch all vulnerabilities: %w", err)
		}
	}

	if len(matchedDeps) > 0 {
		oldestMatchedAt := *matchedDeps[0].LastMatchedAt
		latestVulns, err = m.fetchVulnerabilitiesSince(oldestMatchedAt)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("fetch latest vulnerabilities: %w", err)
		}
	}

	return unmatchedDeps, matchedDeps, allVulns, latestVulns, nil
}

func (m *MatchVulnerabilitiesUseCase) fetchVulnerabilitiesSince(t time.Time) ([]domain.Vulnerability, error) {
	vulns, err := m.vulnerabilityRepo.GetNewerThan(t)
	if err != nil {
		return nil, fmt.Errorf("fetch vulnerabilities since %v: %w", t, err)
	}
	return vulns, nil
}

func splitByMatchedStatus(deps []domain.RepositoryDependency) (unmatched []domain.RepositoryDependency, matched []domain.RepositoryDependency) {
	for _, dep := range deps {
		if dep.LastMatchedAt == nil {
			unmatched = append(unmatched, dep)
		} else {
			matched = append(matched, dep)
		}
	}
	return
}

func findAllMatches(
	unmatchedDeps []domain.RepositoryDependency,
	matchedDeps []domain.RepositoryDependency,
	allVulns []domain.Vulnerability,
	latestVulns []domain.Vulnerability,
	repositoryID string,
) []domain.Match {
	var matches []domain.Match
	matches = append(matches, findMatches(unmatchedDeps, allVulns, repositoryID)...)
	matches = append(matches, findMatches(matchedDeps, latestVulns, repositoryID)...)
	return matches
}

func findMatches(deps []domain.RepositoryDependency, vulns []domain.Vulnerability, repositoryID string) []domain.Match {
	var matches []domain.Match
	for _, vuln := range vulns {
		for _, dep := range deps {
			match, ok := match(vuln, dep, repositoryID)
			if ok {
				matches = append(matches, match)
			}
		}
	}
	return matches
}

func match(vuln domain.Vulnerability, dep domain.RepositoryDependency, repositoryID string) (domain.Match, bool) {
	for _, affected := range vuln.AffectedTechnologies {
		if affected.PURL != "" && dep.PURL != "" {
			if purlsMatch(affected.PURL, dep.PURL) {
				status := confirmOrWarn(affected.VersionRange, dep.Version)
				return buildMatch(vuln, dep, repositoryID, status), true
			}
			return domain.Match{}, false
		}

		if namesMatch(affected.Name, dep.Name) {
			status := confirmOrWarn(affected.VersionRange, dep.Version)
			return buildMatch(vuln, dep, repositoryID, status), true
		}
	}
	return domain.Match{}, false
}

func buildMatch(vuln domain.Vulnerability, dep domain.RepositoryDependency, repositoryID string, status domain.MatchStatus) domain.Match {
	return domain.Match{
		VulnerabilityID:  vuln.ID,
		RepositoryID:     repositoryID,
		MatchedComponent: dep.Name,
		MatchedVersion:   dep.Version,
		ComponentPURL:    dep.PURL,
		Status:           status,
	}
}

func (m *MatchVulnerabilitiesUseCase) persistMatches(repo domain.WatchedRepository, matches []domain.Match) (int, []error) {
	var errs []error
	found := 0

	for _, match := range matches {
		match.ID = domain.NewID()
		if err := m.matchRepository.Save(match); err != nil {
			slog.Error("failed to save match",
				"stage", StageMatch,
				"repo", repo.RepositoryName,
				"vulnerability_id", match.VulnerabilityID,
				"err", err,
			)
			errs = append(errs, fmt.Errorf("%s: save match for %s: %w", StageMatch, repo.RepositoryName, err))
			continue
		}
		found++
	}
	return found, errs
}

func (m *MatchVulnerabilitiesUseCase) updateLastMatched(deps []domain.RepositoryDependency) []error {
	var errs []error
	now := time.Now()

	for _, dep := range deps {
		if err := m.dependencyRepo.UpdateLastMatchedAt(dep.ID, now); err != nil {
			slog.Error("failed to update last_matched_at",
				"stage", StageMatch,
				"dep", dep.Name,
				"err", err,
			)
			errs = append(errs, fmt.Errorf("%s: update last matched for dep %s: %w", StageMatch, dep.Name, err))
		}
	}
	return errs
}

func purlsMatch(a, b string) bool {
	return stripPURLVersion(a) == stripPURLVersion(b)
}

func stripPURLVersion(purl string) string {
	if idx := strings.Index(purl, "@"); idx != -1 {
		return purl[:idx]
	}
	return purl
}

func namesMatch(a, b string) bool {
	nameA := extractPackageName(a)
	nameB := extractPackageName(b)
	ecosystemA := extractEcosystem(a)
	ecosystemB := extractEcosystem(b)
	if ecosystemA != "" && ecosystemB != "" && !strings.EqualFold(ecosystemA, ecosystemB) {
		return false
	}
	return strings.EqualFold(nameA, nameB)
}

func extractPackageName(name string) string {
	p, err := packageurl.FromString(name)
	if err != nil {
		return name
	}
	return p.Name
}

func extractEcosystem(name string) string {
	p, err := packageurl.FromString(name)
	if err != nil {
		return ""
	}
	return p.Type
}

// TODO: version matching needs to be added
func confirmOrWarn(versionRange, version string) domain.MatchStatus {
	if versionRange == "" || version == "" {
		return domain.MatchStatusWarning
	}
	return domain.MatchStatusConfirmed
}
