package handler

import (
	"encoding/json"
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
