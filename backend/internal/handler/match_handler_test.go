package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

type mockMatchDetailRepository struct {
	match *domain.Match
	err   error
}

func (m *mockMatchDetailRepository) GetByID(id string) (*domain.Match, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.match, nil
}

type mockMatchVulnerabilityRepository struct {
	vuln *domain.Vulnerability
	err  error
}

func (m *mockMatchVulnerabilityRepository) GetByID(id string) (*domain.Vulnerability, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.vuln, nil
}

type mockMatchArticleRepository struct {
	articles map[string]*domain.Article
	err      error
}

func (m *mockMatchArticleRepository) GetByID(id string) (*domain.Article, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.articles[id], nil
}

func matchDetailRequest(h http.HandlerFunc, id string) *httptest.ResponseRecorder {
	r := chi.NewRouter()
	r.Get("/matches/{id}", h)
	req := httptest.NewRequest("GET", "/matches/"+id, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestGetMatchDetailHandler_Success(t *testing.T) {
	published := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	match := &domain.Match{
		ID:               "match-1",
		VulnerabilityID:  "vuln-1",
		RepositoryID:     "repo-1",
		ComponentPURL:    "pkg:npm/lodash@4.17.15",
		MatchedComponent: "lodash",
		MatchedVersion:   "4.17.15",
		Status:           domain.MatchStatusConfirmed,
		Confidence:       "CONFIRMED",
		MatchedOn:        "normalized_name",
		VulnIdentifier:   "lodash",
		DepIdentifier:    "lodash",
		VersionCheck:     "in_range",
	}
	vuln := &domain.Vulnerability{
		ID:          "vuln-1",
		CVE:         "CVE-2020-8203",
		Severity:    domain.SeverityHigh,
		Description: "Prototype pollution in lodash",
		AffectedTechnologies: []domain.AffectedTechnology{
			{Name: "lodash", PURL: "pkg:npm/lodash", VersionRange: "<4.17.21"},
		},
		SourceArticleIDs: []string{"article-1", "article-2"},
		PublishedAt:      published,
	}
	articles := map[string]*domain.Article{
		"article-1": {
			ID:          "article-1",
			Headline:    "Lodash vulnerability disclosed",
			SourceURL:   "https://www.bleepingcomputer.com/news/security/lodash",
			HostDomain:  domain.BleepingComputer,
			PublishedAt: published,
		},
		"article-2": {
			ID:          "article-2",
			Headline:    "Second article on lodash",
			SourceURL:   "https://socket.dev/blog/lodash",
			HostDomain:  domain.SocketDev,
			PublishedAt: published,
		},
	}

	handler := GetMatchDetailHandler(
		&mockMatchDetailRepository{match: match},
		&mockMatchVulnerabilityRepository{vuln: vuln},
		&mockMatchArticleRepository{articles: articles},
	)

	w := matchDetailRequest(handler, "match-1")
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	for _, want := range []string{
		`"cve":"CVE-2020-8203"`,
		`"severity":"HIGH"`,
		`"name":"lodash"`,
		`"version":"4.17.15"`,
		`"purl":"pkg:npm/lodash@4.17.15"`,
		`"version_range":"<4.17.21"`,
		`"matched_on":"normalized_name"`,
		`"vuln_identifier":"lodash"`,
		`"dep_identifier":"lodash"`,
		`"confidence":"CONFIRMED"`,
		`"version_check":"in_range"`,
		`"headline":"Lodash vulnerability disclosed"`,
		`"headline":"Second article on lodash"`,
	} {
		if !contains(body, want) {
			t.Errorf("response body should contain %s\ngot: %s", want, body)
		}
	}
}

func TestGetMatchDetailHandler_NotFound(t *testing.T) {
	handler := GetMatchDetailHandler(
		&mockMatchDetailRepository{},
		&mockMatchVulnerabilityRepository{},
		&mockMatchArticleRepository{},
	)

	w := matchDetailRequest(handler, "missing")
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestGetMatchDetailHandler_DatabaseError(t *testing.T) {
	handler := GetMatchDetailHandler(
		&mockMatchDetailRepository{err: fmt.Errorf("db down")},
		&mockMatchVulnerabilityRepository{},
		&mockMatchArticleRepository{},
	)

	w := matchDetailRequest(handler, "match-1")
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestGetMatchDetailHandler_MissingVulnerabilityStillServesMatch(t *testing.T) {
	match := &domain.Match{
		ID:               "match-1",
		VulnerabilityID:  "vuln-missing",
		RepositoryID:     "repo-1",
		ComponentPURL:    "pkg:npm/lodash@4.17.15",
		MatchedComponent: "lodash",
		MatchedVersion:   "4.17.15",
		Status:           domain.MatchStatusWarning,
	}
	handler := GetMatchDetailHandler(
		&mockMatchDetailRepository{match: match},
		&mockMatchVulnerabilityRepository{},
		&mockMatchArticleRepository{},
	)

	w := matchDetailRequest(handler, "match-1")
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if !contains(w.Body.String(), `"matched_component":"lodash"`) {
		t.Errorf("response should still contain the match data, got: %s", w.Body.String())
	}
	if !contains(w.Body.String(), `"source_articles":[]`) {
		t.Errorf("response should contain empty source_articles, got: %s", w.Body.String())
	}
}

func TestGetMatchDetailHandler_VulnerabilityError(t *testing.T) {
	handler := GetMatchDetailHandler(
		&mockMatchDetailRepository{match: &domain.Match{ID: "match-1", VulnerabilityID: "vuln-1"}},
		&mockMatchVulnerabilityRepository{err: fmt.Errorf("db down")},
		&mockMatchArticleRepository{},
	)

	w := matchDetailRequest(handler, "match-1")
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestGetMatchDetailHandler_ArticleError(t *testing.T) {
	handler := GetMatchDetailHandler(
		&mockMatchDetailRepository{match: &domain.Match{ID: "match-1", VulnerabilityID: "vuln-1"}},
		&mockMatchVulnerabilityRepository{vuln: &domain.Vulnerability{
			ID:               "vuln-1",
			SourceArticleIDs: []string{"article-1"},
		}},
		&mockMatchArticleRepository{err: fmt.Errorf("db down")},
	)

	w := matchDetailRequest(handler, "match-1")
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}
