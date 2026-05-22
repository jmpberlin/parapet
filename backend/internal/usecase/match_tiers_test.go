package usecase

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestCheckVersionRange_InRange(t *testing.T) {
	assert.Equal(t, versionInRange, checkVersionRange("<=8.0.0", "7.4.0"))
}

func TestCheckVersionRange_OutOfRange(t *testing.T) {
	assert.Equal(t, versionOutOfRange, checkVersionRange("< v0.38.0", "v0.50.0"))
}

func TestCheckVersionRange_EmptyRange(t *testing.T) {
	assert.Equal(t, versionUnknown, checkVersionRange("", "v1.0.0"))
}

func TestCheckVersionRange_PseudoVersion(t *testing.T) {
	// depNorm.Version is already cleaned before reaching checkVersionRange
	assert.Equal(t, versionInRange, checkVersionRange("< v0.22.0", "v0.21.1"))
}

func TestCheckVersionRange_IncompatibleSuffixOutOfRange(t *testing.T) {
	assert.Equal(t, versionOutOfRange, checkVersionRange("< v28.0.0", "v28.5.2+incompatible"))
}
func TestCheckVersionRange_IncompatibleSuffixInRange(t *testing.T) {
	assert.Equal(t, versionInRange, checkVersionRange("< v28.0.0", "v26.5.2+incompatible"))
}

func TestMatchByPURL_ExactMatch(t *testing.T) {
	vuln := NormalizedIdentifier{Ecosystem: "npm", PackageName: "react"}
	dep := NormalizedIdentifier{Ecosystem: "npm", PackageName: "react"}
	result := matchByPURL(vuln, dep)
	assert.True(t, result.matched)
	assert.Equal(t, ConfidenceHigh, result.confidence)
	assert.Equal(t, "purl", result.matchedOn)
}

func TestMatchByPURL_DifferentEcosystem(t *testing.T) {
	vuln := NormalizedIdentifier{Ecosystem: "npm", PackageName: "decimal"}
	dep := NormalizedIdentifier{Ecosystem: "golang", PackageName: "decimal"}
	result := matchByPURL(vuln, dep)
	assert.False(t, result.matched)
}

func TestMatchByPURL_DifferentName(t *testing.T) {
	vuln := NormalizedIdentifier{Ecosystem: "npm", PackageName: "react"}
	dep := NormalizedIdentifier{Ecosystem: "npm", PackageName: "react-is"}
	result := matchByPURL(vuln, dep)
	assert.False(t, result.matched)
}

func TestMatchByPURL_EmptyEcosystem(t *testing.T) {
	vuln := NormalizedIdentifier{Ecosystem: "", PackageName: "react"}
	dep := NormalizedIdentifier{Ecosystem: "npm", PackageName: "react"}
	result := matchByPURL(vuln, dep)
	assert.False(t, result.matched)
}

func TestMatchByNamespace_ScopedWildcard(t *testing.T) {
	vuln := NormalizedIdentifier{Ecosystem: "npm", Namespace: "tanstack", PackageName: "tanstack"}
	dep := NormalizedIdentifier{Ecosystem: "npm", Namespace: "tanstack", PackageName: "query-core"}
	result := matchByNamespace(vuln, dep)
	assert.True(t, result.matched)
	assert.Equal(t, ConfidenceHigh, result.confidence)
	assert.Equal(t, "namespace", result.matchedOn)
	assert.Equal(t, "tanstack", result.vulnIdentifier)
}

func TestMatchByNamespace_SpecificPackageNotWildcard(t *testing.T) {
	vuln := NormalizedIdentifier{Ecosystem: "npm", Namespace: "tanstack", PackageName: "react-query"}
	dep := NormalizedIdentifier{Ecosystem: "npm", Namespace: "tanstack", PackageName: "query-core"}
	result := matchByNamespace(vuln, dep)
	assert.False(t, result.matched)
}

func TestMatchByNamespace_DifferentNamespace(t *testing.T) {
	vuln := NormalizedIdentifier{Ecosystem: "npm", Namespace: "tanstack", PackageName: "tanstack"}
	dep := NormalizedIdentifier{Ecosystem: "npm", Namespace: "babel", PackageName: "core"}
	result := matchByNamespace(vuln, dep)
	assert.False(t, result.matched)
}

func TestMatchByPackageName_SameNameNoEcosystem(t *testing.T) {
	vuln := NormalizedIdentifier{Ecosystem: "", PackageName: "express"}
	dep := NormalizedIdentifier{Ecosystem: "npm", PackageName: "express"}
	result := matchByPackageName(vuln, dep)
	assert.True(t, result.matched)
	assert.Equal(t, ConfidenceMedium, result.confidence)
}

func TestMatchByPackageName_DifferentEcosystems(t *testing.T) {
	vuln := NormalizedIdentifier{Ecosystem: "pypi", PackageName: "express"}
	dep := NormalizedIdentifier{Ecosystem: "npm", PackageName: "express"}
	result := matchByPackageName(vuln, dep)
	assert.False(t, result.matched)
}

func TestMatchByKeyword_TokenMatch(t *testing.T) {
	vuln := NormalizedIdentifier{Words: []string{"tanstack", "packages"}}
	dep := NormalizedIdentifier{PackageName: "query-core", Words: []string{"tanstack", "query", "core"}}
	result := matchByKeyword(vuln, dep)
	assert.True(t, result.matched)
	assert.Equal(t, ConfidenceLow, result.confidence)
	assert.Equal(t, "word_token", result.matchedOn)
	assert.Equal(t, "tanstack", result.vulnIdentifier)
}

func TestMatchByKeyword_NoOverlap(t *testing.T) {
	vuln := NormalizedIdentifier{Words: []string{"cisco", "firewall"}}
	dep := NormalizedIdentifier{PackageName: "react", Words: []string{"react"}}
	result := matchByKeyword(vuln, dep)
	assert.False(t, result.matched)
}

func TestApplyVersionCheck_Upgraded(t *testing.T) {
	result := tierResult{matched: true, confidence: ConfidenceHigh, matchedOn: "purl"}
	updated := applyVersionCheck(result, "<=8.0.0", "7.4.0")
	assert.Equal(t, ConfidenceConfirmed, updated.confidence)
}

func TestApplyVersionCheck_Downgraded(t *testing.T) {
	result := tierResult{matched: true, confidence: ConfidenceHigh, matchedOn: "purl"}
	updated := applyVersionCheck(result, "< v0.38.0", "v0.50.0")
	assert.Equal(t, ConfidenceLow, updated.confidence)
}

func TestApplyVersionCheck_NoVersionRange(t *testing.T) {
	result := tierResult{matched: true, confidence: ConfidenceHigh, matchedOn: "purl"}
	updated := applyVersionCheck(result, "", "v1.0.0")
	assert.Equal(t, ConfidenceWarning, updated.confidence)
}

func TestApplyVersionCheck_LowConfidenceUnchanged(t *testing.T) {
	result := tierResult{matched: true, confidence: ConfidenceLow, matchedOn: "word_token"}
	updated := applyVersionCheck(result, "", "v1.0.0")
	assert.Equal(t, ConfidenceLow, updated.confidence)
}

func TestMatchAffected_PURLMatch(t *testing.T) {
	affected := domain.AffectedTechnology{
		Name: "react",
		PURL: "pkg:npm/react",
	}
	dep := domain.RepositoryDependency{
		Name:    "react",
		PURL:    "pkg:npm/react@19.2.4",
		Version: "19.2.4",
	}
	match, ok := matchAffected("vuln-1", affected, dep, "repo-1")
	assert.True(t, ok)
	assert.Equal(t, "purl", match.MatchedOn)
	assert.Equal(t, "react", match.MatchedComponent)
}

func TestMatchAffected_WildcardRejected(t *testing.T) {
	affected := domain.AffectedTechnology{
		Name: "Multiple npm packages",
		PURL: "pkg:npm/*",
	}
	dep := domain.RepositoryDependency{
		Name:    "react",
		PURL:    "pkg:npm/react@19.2.4",
		Version: "19.2.4",
	}
	_, ok := matchAffected("vuln-1", affected, dep, "repo-1")
	assert.False(t, ok)
}

func TestMatchAffected_DifferentEcosystemNoMatch(t *testing.T) {
	affected := domain.AffectedTechnology{
		Name:         "decimal",
		PURL:         "pkg:golang/github.com/shopsprint/decimal@v1.3.3",
		VersionRange: "v1.3.3",
	}
	dep := domain.RepositoryDependency{
		Name:    "decimal.js",
		PURL:    "pkg:npm/decimal.js@10.6.0",
		Version: "10.6.0",
	}
	_, ok := matchAffected("vuln-1", affected, dep, "repo-1")
	assert.False(t, ok)
}

func TestMatchAffected_WordTokenMatch(t *testing.T) {
	affected := domain.AffectedTechnology{
		Name: "TanStack packages",
		PURL: "",
	}
	dep := domain.RepositoryDependency{
		Name:    "@tanstack/query-core",
		PURL:    "pkg:npm/%40tanstack/query-core@5.90.20",
		Version: "5.90.20",
	}
	match, ok := matchAffected("vuln-1", affected, dep, "repo-1")
	assert.True(t, ok)
	assert.Equal(t, "word_token", match.MatchedOn)
}

func TestMatchAffected_Fixtures(t *testing.T) {
	type fixture struct {
		Description        string `json:"description"`
		VulnName           string `json:"vuln_name"`
		VulnPURL           string `json:"vuln_purl"`
		VulnVersionRange   string `json:"vuln_version_range"`
		DepName            string `json:"dep_name"`
		DepPURL            string `json:"dep_purl"`
		DepVersion         string `json:"dep_version"`
		ExpectedMatch      bool   `json:"expected_match"`
		ExpectedConfidence string `json:"expected_confidence"`
		ExpectedTier       string `json:"expected_tier"`
		Note               string `json:"note"`
	}

	data, err := os.ReadFile("testdata/matching_fixture.json")
	if err != nil {
		t.Fatalf("failed to read fixture file: %v", err)
	}

	var fixtures []fixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatalf("failed to parse fixture file: %v", err)
	}

	for _, fx := range fixtures {
		t.Run(fx.Description, func(t *testing.T) {
			affected := domain.AffectedTechnology{
				Name:         fx.VulnName,
				PURL:         fx.VulnPURL,
				VersionRange: fx.VulnVersionRange,
			}
			dep := domain.RepositoryDependency{
				Name:    fx.DepName,
				PURL:    fx.DepPURL,
				Version: fx.DepVersion,
			}
			vuln := domain.Vulnerability{
				ID: "test-vuln-id",
			}

			match, ok := matchAffected(vuln.ID, affected, dep, "test-repo-id")

			assert.Equal(t, fx.ExpectedMatch, ok, "match result mismatch — note: %s", fx.Note)

			if fx.ExpectedMatch && ok {
				if fx.ExpectedConfidence != "" {
					assert.Equal(t, fx.ExpectedConfidence, match.Confidence,
						"confidence mismatch for: %s", fx.Description)
				}
				if fx.ExpectedTier != "" {
					assert.Equal(t, fx.ExpectedTier, match.MatchedOn,
						"tier mismatch for: %s", fx.Description)
				}
			}
		})
	}
}
