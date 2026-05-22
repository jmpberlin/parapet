package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalize_ScopedNpmURLEncoded(t *testing.T) {
	n := Normalize("pkg:npm/%40tanstack/query-core@5.90.20", "")
	assert.Equal(t, "npm", n.Ecosystem)
	assert.Equal(t, "query-core", n.PackageName)
	assert.Equal(t, "5.90.20", n.Version)
}

func TestNormalize_GolangPURL(t *testing.T) {
	n := Normalize("pkg:golang/golang.org/x/net@v0.50.0", "")
	assert.Equal(t, "golang", n.Ecosystem)
	assert.Equal(t, "net", n.PackageName)
	assert.Equal(t, "v0.50.0", n.Version)
}

func TestNormalize_GoPseudoVersion(t *testing.T) {
	n := Normalize("pkg:golang/golang.org/x/tools@v0.21.1-0.20240508182429-e35e4ccd0d2d", "")
	assert.Equal(t, "v0.21.1", n.Version)
}

func TestNormalize_GoIncompatible(t *testing.T) {
	n := Normalize("pkg:golang/github.com/docker/docker@v28.5.2+incompatible", "")
	assert.Equal(t, "v28.5.2", n.Version)
}

func TestNormalize_WildcardRejection(t *testing.T) {
	n := Normalize("pkg:npm/*", "")
	assert.Empty(t, n.Ecosystem)
	assert.Empty(t, n.PackageName)
	assert.Empty(t, n.Version)
	assert.Equal(t, "pkg:npm/*", n.RawInput)
}

func TestNormalize_UnknownPlaceholderRejection(t *testing.T) {
	n := Normalize("pkg:npm/<unknown>", "")
	assert.Empty(t, n.Ecosystem)
	assert.Empty(t, n.PackageName)
	assert.Empty(t, n.Version)
}

func TestNormalize_UnknownUppercasePlaceholderRejection(t *testing.T) {
	n := Normalize("pkg:golang/<UNKNOWN>", "")
	assert.Empty(t, n.Ecosystem)
	assert.Empty(t, n.PackageName)
	assert.Empty(t, n.Version)
}

func TestNormalize_ScopedNpmURLEncodedReactQuery(t *testing.T) {
	n := Normalize("pkg:npm/%40tanstack/react-query@5.90.21", "")
	assert.Equal(t, "npm", n.Ecosystem)
	assert.Equal(t, "react-query", n.PackageName)
}

func TestNormalize_PlainNameWords(t *testing.T) {
	n := Normalize("", "TanStack packages")
	assert.Contains(t, n.Words, "tanstack")
}

func TestNormalize_PlainNameGenericFiltering(t *testing.T) {
	n := Normalize("", "React JavaScript library")
	assert.Contains(t, n.Words, "react")
	assert.NotContains(t, n.Words, "javascript")
	assert.NotContains(t, n.Words, "library")
}

func TestNormalize_GoMajorVersionSegmentSkipped(t *testing.T) {
	n := Normalize("pkg:golang/github.com/pressly/goose/v3@v3.27.0", "")
	assert.Equal(t, "goose", n.PackageName)
	assert.Equal(t, "v3.27.0", n.Version)
}

func TestNormalize_ScopedNpmAtSign(t *testing.T) {
	n := Normalize("pkg:npm/@bitwarden/cli@2026.4.0", "")
	assert.Equal(t, "npm", n.Ecosystem)
	assert.Equal(t, "cli", n.PackageName)
}

func TestNormalize_EmptyInputs(t *testing.T) {
	n := Normalize("", "")
	assert.Equal(t, NormalizedIdentifier{}, n)
}

func TestNormalize_ScopedWildcardKept(t *testing.T) {
	n := Normalize("pkg:npm/%40tanstack/*", "")
	assert.Equal(t, "npm", n.Ecosystem)
	assert.Equal(t, "tanstack", n.PackageName)
	assert.Contains(t, n.Words, "tanstack")
}

func TestNormalize_UnscopedWildcardRejected(t *testing.T) {
	n := Normalize("pkg:npm/*", "")
	assert.Empty(t, n.Ecosystem)
	assert.Empty(t, n.PackageName)
}
