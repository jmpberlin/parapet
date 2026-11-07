package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

type mockArticleRepoForTests struct {
	articles  []domain.Article
	total     int
	lastDays  int
	lastPage  int
	lastLimit int
	getErr    error
}

func (m *mockArticleRepoForTests) GetByDaysPaginated(days, page, limit int) ([]domain.Article, int, error) {
	if m.getErr != nil {
		return nil, 0, m.getErr
	}
	m.lastDays = days
	m.lastPage = page
	m.lastLimit = limit
	return m.articles, m.total, nil
}

func (m *mockArticleRepoForTests) GetByID(id string) (*domain.Article, error) {
	return nil, nil
}

func TestGetArticlesHandler_PaginatedResponse(t *testing.T) {
	articles := []domain.Article{
		{ID: "a-1", Headline: "First", PublishedAt: time.Now()},
		{ID: "a-2", Headline: "Second", PublishedAt: time.Now()},
	}
	repo := &mockArticleRepoForTests{articles: articles, total: 12}

	req := httptest.NewRequest(http.MethodGet, "/articles?days=7&page=2&limit=10", nil)
	rec := httptest.NewRecorder()

	GetArticlesHandler(repo).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp paginatedArticlesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %s", err)
	}
	if len(resp.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(resp.Items))
	}
	if resp.Total != 12 {
		t.Errorf("expected total 12, got %d", resp.Total)
	}
	if resp.Page != 2 {
		t.Errorf("expected page 2, got %d", resp.Page)
	}
	if resp.Limit != 10 {
		t.Errorf("expected limit 10, got %d", resp.Limit)
	}
	if resp.TotalPages != 2 {
		t.Errorf("expected 2 total pages, got %d", resp.TotalPages)
	}
}

func TestGetArticlesHandler_DefaultPagination(t *testing.T) {
	repo := &mockArticleRepoForTests{total: 0}

	req := httptest.NewRequest(http.MethodGet, "/articles", nil)
	rec := httptest.NewRecorder()

	GetArticlesHandler(repo).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if repo.lastPage != 1 || repo.lastLimit != 10 {
		t.Errorf("expected page=1 limit=10, got page=%d limit=%d", repo.lastPage, repo.lastLimit)
	}
}

func TestGetArticlesHandler_ClampsExcessiveDays(t *testing.T) {
	repo := &mockArticleRepoForTests{}

	req := httptest.NewRequest(http.MethodGet, "/articles?days=2000000000", nil)
	rec := httptest.NewRecorder()

	GetArticlesHandler(repo).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if repo.lastDays != maxArticleWindowDays {
		t.Errorf("expected days clamped to %d, got %d", maxArticleWindowDays, repo.lastDays)
	}
}

func TestGetArticlesHandler_InvalidPagination(t *testing.T) {
	repo := &mockArticleRepoForTests{}

	req := httptest.NewRequest(http.MethodGet, "/articles?page=abc", nil)
	rec := httptest.NewRecorder()

	GetArticlesHandler(repo).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestGetArticlesHandler_InvalidDays(t *testing.T) {
	repo := &mockArticleRepoForTests{}

	for _, days := range []string{"0", "-5", "abc"} {
		req := httptest.NewRequest(http.MethodGet, "/articles?days="+days, nil)
		rec := httptest.NewRecorder()

		GetArticlesHandler(repo).ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("days=%s: expected 400, got %d", days, rec.Code)
		}
	}
	if repo.lastDays != 0 {
		t.Errorf("repo must not be called with invalid days, got days=%d", repo.lastDays)
	}
}

func TestGetArticlesHandler_DaysWindowBoundaries(t *testing.T) {
	repo := &mockArticleRepoForTests{}

	req := httptest.NewRequest(http.MethodGet, "/articles?days=365", nil)
	rec := httptest.NewRecorder()
	GetArticlesHandler(repo).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || repo.lastDays != 365 {
		t.Errorf("expected 365 to pass through, got code=%d days=%d", rec.Code, repo.lastDays)
	}

	req = httptest.NewRequest(http.MethodGet, "/articles?days=366", nil)
	rec = httptest.NewRecorder()
	GetArticlesHandler(repo).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || repo.lastDays != 365 {
		t.Errorf("expected 366 clamped to 365, got code=%d days=%d", rec.Code, repo.lastDays)
	}
}

func TestGetArticlesHandler_DefaultPageAndWindow(t *testing.T) {
	repo := &mockArticleRepoForTests{}

	req := httptest.NewRequest(http.MethodGet, "/articles", nil)
	rec := httptest.NewRecorder()

	GetArticlesHandler(repo).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if repo.lastDays != 7 {
		t.Errorf("expected default days=7, got %d", repo.lastDays)
	}
}

func TestGetArticlesHandler_PaginationBoundaries(t *testing.T) {
	cases := []struct {
		query     string
		wantCode  int
		wantPage  int
		wantLimit int
	}{
		{query: "page=1&limit=1", wantCode: 200, wantPage: 1, wantLimit: 1},
		{query: "page=1000000&limit=100", wantCode: 200, wantPage: 1000000, wantLimit: 100},
		{query: "page=0", wantCode: 400},
		{query: "page=1000001", wantCode: 400},
		{query: "limit=0", wantCode: 400},
		{query: "limit=101", wantCode: 400},
		{query: "limit=abc", wantCode: 400},
		{query: "page=-3", wantCode: 400},
	}

	for _, tc := range cases {
		repo := &mockArticleRepoForTests{}
		req := httptest.NewRequest(http.MethodGet, "/articles?"+tc.query, nil)
		rec := httptest.NewRecorder()

		GetArticlesHandler(repo).ServeHTTP(rec, req)

		if rec.Code != tc.wantCode {
			t.Errorf("%s: expected %d, got %d", tc.query, tc.wantCode, rec.Code)
			continue
		}
		if tc.wantCode != 200 {
			continue
		}
		if repo.lastPage != tc.wantPage || repo.lastLimit != tc.wantLimit {
			t.Errorf("%s: expected page=%d limit=%d, got page=%d limit=%d",
				tc.query, tc.wantPage, tc.wantLimit, repo.lastPage, repo.lastLimit)
		}
	}
}

func TestGetArticlesHandler_RepoError(t *testing.T) {
	repo := &mockArticleRepoForTests{getErr: errors.New("db down")}

	req := httptest.NewRequest(http.MethodGet, "/articles", nil)
	rec := httptest.NewRecorder()

	GetArticlesHandler(repo).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestGetArticlesHandler_EmptyWindowTotalPagesIsOne(t *testing.T) {
	repo := &mockArticleRepoForTests{articles: []domain.Article{}, total: 0}

	req := httptest.NewRequest(http.MethodGet, "/articles", nil)
	rec := httptest.NewRecorder()

	GetArticlesHandler(repo).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp paginatedArticlesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %s", err)
	}
	if resp.Items == nil {
		t.Error("expected items to serialize as an empty array, got null")
	}
	if resp.TotalPages != 1 {
		t.Errorf("expected total_pages 1 for an empty result, got %d", resp.TotalPages)
	}
}
