package usecase

// Matching a vulnerability against a dependency works in four tiers, tried in order.
// The first tier that finds a hit wins; lower tiers are only reached when higher ones fail.
//
//  Tier 1 — PURL match        exact ecosystem + package name            → HIGH confidence
//  Tier 2 — Namespace match   same ecosystem + namespace wildcard       → HIGH confidence
//  Tier 3 — Package name      exact name, ecosystem optional            → MEDIUM confidence
//  Tier 4 — Keyword match     shared search terms                       → LOW confidence
//
// Once a hit is found, the vulnerability's version range is checked to adjust
// the final confidence:
//
//  dependency IS in vulnerable range   → CONFIRMED
//  dependency is OUTSIDE range         → LOW  (packages match but this version isn't affected)
//  no version data available    		→ WARNING for HIGH/MEDIUM hits, LOW stays LOW

import (
	"strings"
	"time"

	semver "github.com/Masterminds/semver/v3"
	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

type MatchConfidence string

const (
	ConfidenceHigh      MatchConfidence = "HIGH"
	ConfidenceMedium    MatchConfidence = "MEDIUM"
	ConfidenceLow       MatchConfidence = "LOW"
	ConfidenceWarning   MatchConfidence = "WARNING"
	ConfidenceConfirmed MatchConfidence = "CONFIRMED"
	ConfidenceNone      MatchConfidence = "NONE"
)

type tierResult struct {
	matched        bool
	confidence     MatchConfidence
	matchedOn      string
	vulnIdentifier string
	depIdentifier  string
}

// matchAffected checks whether a single affected technology entry from a
// vulnerability matches a repository dependency
func matchAffected(vulnID string, affected domain.AffectedTechnology, dep domain.RepositoryDependency, repositoryID string,
) (domain.Match, bool) {
	vulnNorm := Normalize(affected.PURL, affected.Name)
	depNorm := Normalize(dep.PURL, dep.Name)

	if vulnNorm.PackageName == "" && len(vulnNorm.Words) == 0 {
		return domain.Match{}, false
	}

	if result := matchByPURL(vulnNorm, depNorm); result.matched {
		result = applyVersionCheck(result, affected.VersionRange, depNorm.Version)
		return buildMatchFromResult(vulnID, dep, repositoryID, result), true
	}
	if result := matchByNamespace(vulnNorm, depNorm); result.matched {
		result = applyVersionCheck(result, affected.VersionRange, depNorm.Version)
		return buildMatchFromResult(vulnID, dep, repositoryID, result), true
	}
	if result := matchByPackageName(vulnNorm, depNorm); result.matched {
		result = applyVersionCheck(result, affected.VersionRange, depNorm.Version)
		return buildMatchFromResult(vulnID, dep, repositoryID, result), true
	}
	if keywordMatchAllowed(vulnNorm, depNorm) {
		if result := matchByKeyword(vulnNorm, depNorm); result.matched {
			result = applyVersionCheck(result, affected.VersionRange, depNorm.Version)
			return buildMatchFromResult(vulnID, dep, repositoryID, result), true
		}
	}

	return domain.Match{}, false
}

// matchByPURL matches when both sides have the same ecosystem and package name.
// This is the most precise tier — it requires a well-formed PURL on both sides.
func matchByPURL(vuln, dep NormalizedIdentifier) tierResult {
	if vuln.Ecosystem == "" || vuln.PackageName == "" {
		return tierResult{}
	}
	if dep.Ecosystem == "" || dep.PackageName == "" {
		return tierResult{}
	}
	if vuln.Ecosystem != dep.Ecosystem || vuln.PackageName != dep.PackageName {
		return tierResult{}
	}
	return tierResult{
		matched:        true,
		confidence:     ConfidenceHigh,
		matchedOn:      "purl",
		vulnIdentifier: vuln.PackageName,
		depIdentifier:  dep.PackageName,
	}
}

// matchByNamespace matches when the vulnerability is a namespace wildcard (e.g.
// pkg:npm/@tanstack/*) and the dependency belongs to that same namespace.
func matchByNamespace(vuln, dep NormalizedIdentifier) tierResult {
	if vuln.Namespace == "" || dep.Namespace == "" {
		return tierResult{}
	}
	if vuln.Ecosystem == "" || dep.Ecosystem == "" || vuln.Ecosystem != dep.Ecosystem {
		return tierResult{}
	}
	if vuln.Namespace != dep.Namespace {
		return tierResult{}
	}
	if vuln.PackageName != vuln.Namespace {
		return tierResult{}
	}
	return tierResult{
		matched:        true,
		confidence:     ConfidenceHigh,
		matchedOn:      "namespace",
		vulnIdentifier: vuln.Namespace,
		depIdentifier:  dep.PackageName,
	}
}

// matchByPackageName matches when both sides share the same package name.
func matchByPackageName(vuln, dep NormalizedIdentifier) tierResult {
	if vuln.PackageName == "" || dep.PackageName == "" {
		return tierResult{}
	}
	if vuln.Ecosystem != "" && dep.Ecosystem != "" && vuln.Ecosystem != dep.Ecosystem {
		return tierResult{}
	}
	if vuln.PackageName != dep.PackageName {
		return tierResult{}
	}
	return tierResult{
		matched:        true,
		confidence:     ConfidenceMedium,
		matchedOn:      "normalized_name",
		vulnIdentifier: vuln.PackageName,
		depIdentifier:  dep.PackageName,
	}
}

// matchByKeyword matches when any keyword from the vulnerability appears in the
// dep's keyword set.
func matchByKeyword(vuln, dep NormalizedIdentifier) tierResult {
	if len(vuln.Words) == 0 || len(dep.Words) == 0 {
		return tierResult{}
	}
	depKeywords := make(map[string]bool, len(dep.Words))
	for _, w := range dep.Words {
		depKeywords[w] = true
	}
	for _, w := range vuln.Words {
		if depKeywords[w] {
			return tierResult{
				matched:        true,
				confidence:     ConfidenceLow,
				matchedOn:      "word_token",
				vulnIdentifier: w,
				depIdentifier:  dep.PackageName,
			}
		}
	}
	return tierResult{}
}

// keywordMatchAllowed prevents two classes of false positives:
//
//   - Different ecosystems: "decimal" (golang) must not match "decimal.js" (npm).
//   - Both sides fully identified in the same ecosystem but with different package
//     names: "react" must not match "react-is" via the shared keyword "react",
//     and "@tanstack/react-router" must not match "@tanstack/react-query" via the
//     shared scope keyword "tanstack".
//
// Exception: if the vulnerability's package name is a generic word (e.g. "packages"
// from a malformed PURL) the vuln has no precise identity, so keyword matching is
// still meaningful.
func keywordMatchAllowed(vuln, dep NormalizedIdentifier) bool {
	if vuln.Ecosystem != "" && dep.Ecosystem != "" && vuln.Ecosystem != dep.Ecosystem {
		return false
	}
	if vuln.Ecosystem != "" && vuln.Ecosystem == dep.Ecosystem &&
		vuln.PackageName != "" && dep.PackageName != "" &&
		!genericTokens[vuln.PackageName] {
		return false
	}
	return true
}

type versionCheckResult int

const (
	versionUnknown versionCheckResult = iota
	versionInRange
	versionOutOfRange
)

// applyVersionCheck adjusts the confidence of a tier hit based on version matching
func applyVersionCheck(result tierResult, vulnVersionRange, depVersion string) tierResult {
	switch checkVersionRange(vulnVersionRange, depVersion) {
	case versionInRange:
		result.confidence = ConfidenceConfirmed
	case versionOutOfRange:
		result.confidence = ConfidenceLow
	default:
		if result.confidence == ConfidenceHigh || result.confidence == ConfidenceMedium {
			result.confidence = ConfidenceWarning
		}
	}
	return result
}

// buildMatchFromResult assembles a domain.Match from the matched dependency and the tier result.
func buildMatchFromResult(
	vulnID string,
	dep domain.RepositoryDependency,
	repositoryID string,
	result tierResult,
) domain.Match {
	status := domain.MatchStatusWarning
	switch result.confidence {
	case ConfidenceConfirmed, ConfidenceHigh, ConfidenceMedium:
		status = domain.MatchStatusConfirmed
	}
	return domain.Match{
		VulnerabilityID:  vulnID,
		RepositoryID:     repositoryID,
		MatchedComponent: dep.Name,
		MatchedVersion:   dep.Version,
		ComponentPURL:    dep.PURL,
		CreatedAt:        time.Now(),
		Status:           status,
		Confidence:       string(result.confidence),
		MatchedOn:        result.matchedOn,
		VulnIdentifier:   result.vulnIdentifier,
		DepIdentifier:    result.depIdentifier,
	}
}

// checkVersionRange reports whether depVersion falls inside or outside the vulnerability's version range
func checkVersionRange(vulnVersionRange, depVersion string) versionCheckResult {
	if vulnVersionRange == "" || depVersion == "" {
		return versionUnknown
	}

	vulnVersionRange = strings.TrimLeft(vulnVersionRange, "=")

	constraint, err := semver.NewConstraint(vulnVersionRange)
	if err != nil {
		return versionUnknown
	}
	version, err := semver.NewVersion(depVersion)
	if err != nil {
		return versionUnknown
	}

	if constraint.Check(version) {
		return versionInRange
	}
	return versionOutOfRange
}
