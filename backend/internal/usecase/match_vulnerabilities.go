package usecase

import (
	"strings"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
	"github.com/jmpberlin/nightwatch/backend/internal/utility"
)

func MatchVulnerabilities(vulnerabilities []domain.Vulnerability, technologies []domain.RepositoryDependency, repositoryID string) []domain.Match {
	var matches []domain.Match

	for _, vuln := range vulnerabilities {
		for _, tech := range technologies {
			match, ok := matchVulnerabilityToTechnology(vuln, tech, repositoryID)
			if ok {
				matches = append(matches, match)
			}
		}
	}
	return matches
}

func matchVulnerabilityToTechnology(vuln domain.Vulnerability, tech domain.RepositoryDependency, repositoryID string) (domain.Match, bool) {
	for _, affected := range vuln.AffectedTechnologies {
		if *affected.PURL != "" && tech.PURL != "" {
			if purlsMatch(*affected.PURL, tech.PURL) {
				status := confirmOrWarn(affected.VersionRange, tech.Version)
				return buildMatch(vuln, tech, repositoryID, status), true
			}
		}
		if namesMatch(affected.Name, tech.Name) {
			status := confirmOrWarn(affected.VersionRange, tech.Version)
			return buildMatch(vuln, tech, repositoryID, status), true
		}
	}
	return domain.Match{}, false
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
	ecosystemA := utility.ExtractEcosystem(a)
	ecosystemB := utility.ExtractEcosystem(b)
	if ecosystemA != "" && ecosystemB != "" && !strings.EqualFold(ecosystemA, ecosystemB) {
		return false
	}
	return strings.EqualFold(nameA, nameB)
}

func extractPackageName(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '/' || r == ':' || r == '@'
	})
	if len(parts) == 0 {
		return name
	}
	return parts[len(parts)-1]
}

func confirmOrWarn(versionRange, version string) domain.MatchStatus {
	if versionRange == "" || version == "" {
		return domain.MatchStatusWarning
	}
	// TODO: semver version range check
	return domain.MatchStatusConfirmed
}

func buildMatch(
	vuln domain.Vulnerability,
	tech domain.RepositoryDependency,
	repositoryID string,
	status domain.MatchStatus,
) domain.Match {
	return domain.Match{
		VulnerabilityID:  vuln.ID,
		RepositoryID:     repositoryID,
		MatchedComponent: tech.Name,
		MatchedVersion:   tech.Version,
		Status:           status,
	}
}
