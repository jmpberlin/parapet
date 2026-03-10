package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

const apiBaseURL = "https://api.github.com/repos"

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type sbomPackage struct {
	Name         string        `json:"name"`
	Version      string        `json:"versionInfo"`
	ExternalRefs []externalRef `json:"externalRefs"`
}

type externalRef struct {
	ReferenceType    string `json:"referenceType"`
	ReferenceLocator string `json:"referenceLocator"`
}

type sbomResponse struct {
	Sbom struct {
		Packages []sbomPackage `json:"packages"`
	} `json:"sbom"`
}

type GitHubClient struct {
	httpClient HTTPDoer
}

func NewGithubClient() *GitHubClient {
	return &GitHubClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (g *GitHubClient) GetDependencies(owner, repo, token string) ([]domain.RepositoryDependency, error) {
	sbom, err := g.fetchSBOM(owner, repo, token)
	if err != nil {
		return nil, err
	}

	// Skipping first package, it's the repository itself
	var dependencies []domain.RepositoryDependency
	for _, pkg := range sbom.Sbom.Packages[1:] {
		dependencies = append(dependencies, domain.RepositoryDependency{
			Name:    pkg.Name,
			Version: pkg.Version,
			PURL:    extractPURL(pkg.ExternalRefs),
		})
	}
	return dependencies, nil
}

func (g *GitHubClient) fetchSBOM(owner, repo, token string) (sbomResponse, error) {
	url := fmt.Sprintf("%s/%s/%s/dependency-graph/sbom", apiBaseURL, owner, repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return sbomResponse{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	res, err := g.httpClient.Do(req)
	if err != nil {
		return sbomResponse{}, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return sbomResponse{}, fmt.Errorf("github API returned %d", res.StatusCode)
	}

	var response sbomResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return sbomResponse{}, fmt.Errorf("failed to decode response: %w", err)
	}
	return response, nil
}

func extractPURL(refs []externalRef) string {
	for _, ref := range refs {
		if ref.ReferenceType == "purl" {
			return ref.ReferenceLocator
		}
	}
	return ""
}
