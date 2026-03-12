package usecase

import (
	"testing"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

func TestDiffDependencies_NewDepWithPURL(t *testing.T) {
	githubDeps := []domain.RepositoryDependency{
		{Name: "react", PURL: "pkg:npm/react@18.0.0"},
	}
	storedDeps := []domain.RepositoryDependency{}

	newDeps, removedIDs := diffDependencies(githubDeps, storedDeps, "repo-1")

	if len(newDeps) != 1 {
		t.Errorf("expected 1 new dep, got %d", len(newDeps))
	}
	if newDeps[0].PURL != "pkg:npm/react@18.0.0" {
		t.Errorf("expected react PURL, got %s", newDeps[0].PURL)
	}
	if newDeps[0].RepositoryID != "repo-1" {
		t.Errorf("expected repo-1, got %s", newDeps[0].RepositoryID)
	}
	if newDeps[0].ID == "" {
		t.Error("expected new dep to have an ID assigned")
	}
	if len(removedIDs) != 0 {
		t.Errorf("expected no removed deps, got %d", len(removedIDs))
	}
}

func TestDiffDependencies_NewDepWithoutPURL(t *testing.T) {
	githubDeps := []domain.RepositoryDependency{
		{Name: "some-internal-lib", PURL: ""},
	}
	storedDeps := []domain.RepositoryDependency{}

	newDeps, removedIDs := diffDependencies(githubDeps, storedDeps, "repo-1")

	if len(newDeps) != 1 {
		t.Errorf("expected 1 new dep, got %d", len(newDeps))
	}
	if newDeps[0].Name != "some-internal-lib" {
		t.Errorf("expected some-internal-lib, got %s", newDeps[0].Name)
	}
	if len(removedIDs) != 0 {
		t.Errorf("expected no removed deps, got %d", len(removedIDs))
	}
}

func TestDiffDependencies_RemovedDepWithPURL(t *testing.T) {
	githubDeps := []domain.RepositoryDependency{}
	storedDeps := []domain.RepositoryDependency{
		{ID: "stored-1", Name: "react", PURL: "pkg:npm/react@18.0.0"},
	}

	newDeps, removedIDs := diffDependencies(githubDeps, storedDeps, "repo-1")

	if len(newDeps) != 0 {
		t.Errorf("expected no new deps, got %d", len(newDeps))
	}
	if len(removedIDs) != 1 {
		t.Errorf("expected 1 removed dep, got %d", len(removedIDs))
	}
	if removedIDs[0] != "stored-1" {
		t.Errorf("expected stored-1 to be removed, got %s", removedIDs[0])
	}
}

func TestDiffDependencies_RemovedDepWithoutPURL(t *testing.T) {
	githubDeps := []domain.RepositoryDependency{}
	storedDeps := []domain.RepositoryDependency{
		{ID: "stored-1", Name: "some-internal-lib", PURL: ""},
	}

	newDeps, removedIDs := diffDependencies(githubDeps, storedDeps, "repo-1")

	if len(newDeps) != 0 {
		t.Errorf("expected no new deps, got %d", len(newDeps))
	}
	if len(removedIDs) != 1 {
		t.Errorf("expected 1 removed dep, got %d", len(removedIDs))
	}
	if removedIDs[0] != "stored-1" {
		t.Errorf("expected stored-1 to be removed, got %s", removedIDs[0])
	}
}

func TestDiffDependencies_UnchangedDep_NotInNewOrRemoved(t *testing.T) {
	githubDeps := []domain.RepositoryDependency{
		{Name: "react", PURL: "pkg:npm/react@18.0.0"},
	}
	storedDeps := []domain.RepositoryDependency{
		{ID: "stored-1", Name: "react", PURL: "pkg:npm/react@18.0.0"},
	}

	newDeps, removedIDs := diffDependencies(githubDeps, storedDeps, "repo-1")

	if len(newDeps) != 0 {
		t.Errorf("expected no new deps, got %d", len(newDeps))
	}
	if len(removedIDs) != 0 {
		t.Errorf("expected no removed deps, got %d", len(removedIDs))
	}
}

func TestDiffDependencies_MixedChanges(t *testing.T) {
	githubDeps := []domain.RepositoryDependency{
		{Name: "react", PURL: "pkg:npm/react@18.0.0"},
		{Name: "lodash", PURL: "pkg:npm/lodash@4.0.0"},
	}
	storedDeps := []domain.RepositoryDependency{
		{ID: "stored-1", Name: "react", PURL: "pkg:npm/react@18.0.0"},
		{ID: "stored-2", Name: "express", PURL: "pkg:npm/express@4.0.0"},
	}

	newDeps, removedIDs := diffDependencies(githubDeps, storedDeps, "repo-1")

	if len(newDeps) != 1 {
		t.Errorf("expected 1 new dep, got %d", len(newDeps))
	}
	if newDeps[0].PURL != "pkg:npm/lodash@4.0.0" {
		t.Errorf("expected lodash to be new, got %s", newDeps[0].PURL)
	}
	if len(removedIDs) != 1 {
		t.Errorf("expected 1 removed dep, got %d", len(removedIDs))
	}
	if removedIDs[0] != "stored-2" {
		t.Errorf("expected express to be removed, got %s", removedIDs[0])
	}
}

func TestDiffDependencies_EmptyBoth_NoChanges(t *testing.T) {
	newDeps, removedIDs := diffDependencies(
		[]domain.RepositoryDependency{},
		[]domain.RepositoryDependency{},
		"repo-1",
	)

	if len(newDeps) != 0 {
		t.Errorf("expected no new deps, got %d", len(newDeps))
	}
	if len(removedIDs) != 0 {
		t.Errorf("expected no removed deps, got %d", len(removedIDs))
	}
}

func TestDiffDependencies_SamePURLDifferentName_TreatedAsUnchanged(t *testing.T) {
	githubDeps := []domain.RepositoryDependency{
		{Name: "react-renamed", PURL: "pkg:npm/react@18.0.0"},
	}
	storedDeps := []domain.RepositoryDependency{
		{ID: "stored-1", Name: "react", PURL: "pkg:npm/react@18.0.0"},
	}

	newDeps, removedIDs := diffDependencies(githubDeps, storedDeps, "repo-1")

	if len(newDeps) != 0 {
		t.Errorf("expected no new deps — PURL match should win over name, got %d", len(newDeps))
	}
	if len(removedIDs) != 0 {
		t.Errorf("expected no removed deps — PURL match should win over name, got %d", len(removedIDs))
	}
}
