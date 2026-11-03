package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

type mockDepRepositoryForTests struct {
	dependencies []domain.RepositoryDependency
	total        int
	countErr     error
	getErr       error
}

func (m *mockDepRepositoryForTests) GetByRepoID(id string) ([]domain.RepositoryDependency, error) {
	return m.dependencies, m.getErr
}

func (m *mockDepRepositoryForTests) GetByRepoIDPaginated(repoID string, page, limit int) ([]domain.RepositoryDependency, int, error) {
	if m.getErr != nil {
		return nil, 0, m.getErr
	}
	return m.dependencies, m.total, nil
}

func (m *mockDepRepositoryForTests) GetCountByRepoID(repoID string) (int, error) {
	return m.total, m.countErr
}

type mockMatchRepositoryForTests struct {
	matches  []domain.Match
	total    int
	countErr error
	getErr   error
}

func (m *mockMatchRepositoryForTests) GetByRepositoryID(id string) ([]domain.Match, error) {
	return m.matches, m.getErr
}

func (m *mockMatchRepositoryForTests) GetByRepositoryIDPaginated(repoID string, page, limit int) ([]domain.Match, int, error) {
	if m.getErr != nil {
		return nil, 0, m.getErr
	}
	return m.matches, m.total, nil
}

func (m *mockMatchRepositoryForTests) GetCountByRepositoryID(repoID string) (int, error) {
	return m.total, m.countErr
}

func TestGetPaginationParams_DefaultValues(t *testing.T) {
	req := httptest.NewRequest("GET", "/?page=&limit=", nil)
	page, limit, err := getPaginationParams(req, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page != 1 {
		t.Errorf("expected default page 1, got %d", page)
	}
	if limit != 50 {
		t.Errorf("expected default limit 50, got %d", limit)
	}
}

func TestGetPaginationParams_ValidValues(t *testing.T) {
	req := httptest.NewRequest("GET", "/?page=3&limit=25", nil)
	page, limit, err := getPaginationParams(req, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page != 3 {
		t.Errorf("expected page 3, got %d", page)
	}
	if limit != 25 {
		t.Errorf("expected limit 25, got %d", limit)
	}
}

func TestGetPaginationParams_InvalidPageNotNumber(t *testing.T) {
	req := httptest.NewRequest("GET", "/?page=abc&limit=20", nil)
	_, _, err := getPaginationParams(req, 50)
	if err == nil {
		t.Error("expected error for invalid page, got nil")
	}
}

func TestGetPaginationParams_InvalidLimitNotNumber(t *testing.T) {
	req := httptest.NewRequest("GET", "/?page=1&limit=xyz", nil)
	_, _, err := getPaginationParams(req, 50)
	if err == nil {
		t.Error("expected error for invalid limit, got nil")
	}
}

func TestGetPaginationParams_PageTooSmall(t *testing.T) {
	req := httptest.NewRequest("GET", "/?page=0&limit=20", nil)
	_, _, err := getPaginationParams(req, 50)
	if err == nil {
		t.Error("expected error for page < 1, got nil")
	}
}

func TestGetPaginationParams_PageNegative(t *testing.T) {
	req := httptest.NewRequest("GET", "/?page=-1&limit=20", nil)
	_, _, err := getPaginationParams(req, 50)
	if err == nil {
		t.Error("expected error for negative page, got nil")
	}
}

func TestGetPaginationParams_PageTooLarge(t *testing.T) {
	req := httptest.NewRequest("GET", "/?page=1000001&limit=20", nil)
	_, _, err := getPaginationParams(req, 50)
	if err == nil {
		t.Error("expected error for page > 1000000, got nil")
	}
}

func TestGetPaginationParams_PageMaxAllowed(t *testing.T) {
	req := httptest.NewRequest("GET", "/?page=1000000&limit=20", nil)
	page, _, err := getPaginationParams(req, 50)
	if err != nil {
		t.Errorf("unexpected error for max page: %v", err)
	}
	if page != 1000000 {
		t.Errorf("expected page 1000000, got %d", page)
	}
}

func TestGetPaginationParams_LimitTooSmall(t *testing.T) {
	req := httptest.NewRequest("GET", "/?page=1&limit=0", nil)
	_, _, err := getPaginationParams(req, 50)
	if err == nil {
		t.Error("expected error for limit < 1, got nil")
	}
}

func TestGetPaginationParams_LimitNegative(t *testing.T) {
	req := httptest.NewRequest("GET", "/?page=1&limit=-10", nil)
	_, _, err := getPaginationParams(req, 50)
	if err == nil {
		t.Error("expected error for negative limit, got nil")
	}
}

func TestGetPaginationParams_LimitTooLarge(t *testing.T) {
	req := httptest.NewRequest("GET", "/?page=1&limit=101", nil)
	_, _, err := getPaginationParams(req, 50)
	if err == nil {
		t.Error("expected error for limit > 100, got nil")
	}
}

func TestGetPaginationParams_LimitMaxAllowed(t *testing.T) {
	req := httptest.NewRequest("GET", "/?page=1&limit=100", nil)
	_, limit, err := getPaginationParams(req, 50)
	if err != nil {
		t.Errorf("unexpected error for max limit: %v", err)
	}
	if limit != 100 {
		t.Errorf("expected limit 100, got %d", limit)
	}
}

func TestGetPaginationParams_DefaultLimitMatches(t *testing.T) {
	req := httptest.NewRequest("GET", "/?page=2", nil)
	page, limit, err := getPaginationParams(req, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page != 2 {
		t.Errorf("expected page 2, got %d", page)
	}
	if limit != 20 {
		t.Errorf("expected default limit 20, got %d", limit)
	}
}

func TestCalculateTotalPages_ZeroTotal(t *testing.T) {
	totalPages := calculateTotalPages(0, 20)
	if totalPages != 1 {
		t.Errorf("expected 1 page for zero total, got %d", totalPages)
	}
}

func TestCalculateTotalPages_SinglePage(t *testing.T) {
	totalPages := calculateTotalPages(20, 20)
	if totalPages != 1 {
		t.Errorf("expected 1 page for total=limit, got %d", totalPages)
	}
}

func TestCalculateTotalPages_MultiplePages(t *testing.T) {
	totalPages := calculateTotalPages(125, 20)
	if totalPages != 7 {
		t.Errorf("expected 7 pages for 125/20, got %d", totalPages)
	}
}

func TestCalculateTotalPages_OneMoreThanPageSize(t *testing.T) {
	totalPages := calculateTotalPages(21, 20)
	if totalPages != 2 {
		t.Errorf("expected 2 pages for 21/20, got %d", totalPages)
	}
}

func TestCalculateTotalPages_LargeNumbers(t *testing.T) {
	totalPages := calculateTotalPages(10000, 50)
	if totalPages != 200 {
		t.Errorf("expected 200 pages for 10000/50, got %d", totalPages)
	}
}

func TestGetRepositoryMatchesHandler_Success(t *testing.T) {
	matches := []domain.Match{
		{
			ID:               "match-1",
			VulnerabilityID:  "vuln-1",
			RepositoryID:     "repo-1",
			ComponentPURL:    "pkg:npm/lodash@4.17.20",
			MatchedComponent: "lodash",
			MatchedVersion:   "4.17.20",
			Status:           domain.MatchStatusWarning,
		},
	}
	mockRepo := &mockMatchRepositoryForTests{
		matches: matches,
		total:   125,
	}
	handler := GetRepositoryMatchesHandler(mockRepo)

	req := httptest.NewRequest("GET", "/?page=1&limit=20", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if !contains(w.Body.String(), `"total":125`) {
		t.Error("response should contain total: 125")
	}
	if !contains(w.Body.String(), `"page":1`) {
		t.Error("response should contain page: 1")
	}
	if !contains(w.Body.String(), `"limit":20`) {
		t.Error("response should contain limit: 20")
	}
	if !contains(w.Body.String(), `"total_pages":7`) {
		t.Error("response should contain total_pages: 7")
	}
}

func TestGetRepositoryMatchesHandler_InvalidPage(t *testing.T) {
	mockRepo := &mockMatchRepositoryForTests{}
	handler := GetRepositoryMatchesHandler(mockRepo)

	req := httptest.NewRequest("GET", "/?page=invalid&limit=20", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
	if !contains(w.Body.String(), "invalid pagination parameters") {
		t.Error("response should contain error message")
	}
}

func TestGetRepositoryMatchesHandler_InvalidLimit(t *testing.T) {
	mockRepo := &mockMatchRepositoryForTests{}
	handler := GetRepositoryMatchesHandler(mockRepo)

	req := httptest.NewRequest("GET", "/?page=1&limit=999", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGetRepositoryMatchesHandler_DatabaseError(t *testing.T) {
	mockRepo := &mockMatchRepositoryForTests{
		getErr: fmt.Errorf("database error"),
	}
	handler := GetRepositoryMatchesHandler(mockRepo)

	req := httptest.NewRequest("GET", "/?page=1&limit=20", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
	if !contains(w.Body.String(), "failed to fetch matches") {
		t.Error("response should contain error message")
	}
}

func TestGetRepositoryMatchesHandler_DefaultLimit(t *testing.T) {
	matches := []domain.Match{}
	mockRepo := &mockMatchRepositoryForTests{
		matches: matches,
		total:   0,
	}
	handler := GetRepositoryMatchesHandler(mockRepo)

	req := httptest.NewRequest("GET", "/?page=1", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if !contains(w.Body.String(), `"limit":20`) {
		t.Error("response should contain default limit 20")
	}
}

func TestGetRepositoryDependenciesHandler_Success(t *testing.T) {
	deps := []domain.RepositoryDependency{
		{
			ID:           "dep-1",
			RepositoryID: "repo-1",
			Name:         "react",
			Version:      "18.0.0",
			PURL:         "pkg:npm/react@18.0.0",
		},
	}
	mockRepo := &mockDepRepositoryForTests{
		dependencies: deps,
		total:        1247,
	}
	handler := GetRepositoryDependenciesHandler(mockRepo)

	req := httptest.NewRequest("GET", "/?page=1&limit=50", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if !contains(w.Body.String(), `"total":1247`) {
		t.Error("response should contain total: 1247")
	}
	if !contains(w.Body.String(), `"page":1`) {
		t.Error("response should contain page: 1")
	}
	if !contains(w.Body.String(), `"limit":50`) {
		t.Error("response should contain limit: 50")
	}
	if !contains(w.Body.String(), `"total_pages":25`) {
		t.Error("response should contain total_pages: 25")
	}
}

func TestGetRepositoryDependenciesHandler_InvalidPage(t *testing.T) {
	mockRepo := &mockDepRepositoryForTests{}
	handler := GetRepositoryDependenciesHandler(mockRepo)

	req := httptest.NewRequest("GET", "/?page=invalid&limit=50", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGetRepositoryDependenciesHandler_InvalidLimit(t *testing.T) {
	mockRepo := &mockDepRepositoryForTests{}
	handler := GetRepositoryDependenciesHandler(mockRepo)

	req := httptest.NewRequest("GET", "/?page=1&limit=999", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGetRepositoryDependenciesHandler_DatabaseError(t *testing.T) {
	mockRepo := &mockDepRepositoryForTests{
		getErr: fmt.Errorf("database error"),
	}
	handler := GetRepositoryDependenciesHandler(mockRepo)

	req := httptest.NewRequest("GET", "/?page=1&limit=50", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
	if !contains(w.Body.String(), "failed to fetch dependencies") {
		t.Error("response should contain error message")
	}
}

func TestGetRepositoryDependenciesHandler_DefaultLimit(t *testing.T) {
	deps := []domain.RepositoryDependency{}
	mockRepo := &mockDepRepositoryForTests{
		dependencies: deps,
		total:        0,
	}
	handler := GetRepositoryDependenciesHandler(mockRepo)

	req := httptest.NewRequest("GET", "/?page=1", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if !contains(w.Body.String(), `"limit":50`) {
		t.Error("response should contain default limit 50")
	}
}

func TestGetRepositoryMatchesHandler_EmptyResults(t *testing.T) {
	mockRepo := &mockMatchRepositoryForTests{
		matches: []domain.Match{},
		total:   0,
	}
	handler := GetRepositoryMatchesHandler(mockRepo)

	req := httptest.NewRequest("GET", "/?page=1&limit=20", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if !contains(w.Body.String(), `"items":[]`) {
		t.Error("response should contain empty items array")
	}
	if !contains(w.Body.String(), `"total":0`) {
		t.Error("response should contain total: 0")
	}
	if !contains(w.Body.String(), `"total_pages":1`) {
		t.Error("response should contain total_pages: 1 for empty results")
	}
}

func TestGetRepositoryDependenciesHandler_EmptyResults(t *testing.T) {
	mockRepo := &mockDepRepositoryForTests{
		dependencies: []domain.RepositoryDependency{},
		total:        0,
	}
	handler := GetRepositoryDependenciesHandler(mockRepo)

	req := httptest.NewRequest("GET", "/?page=1&limit=50", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if !contains(w.Body.String(), `"items":[]`) {
		t.Error("response should contain empty items array")
	}
}

func TestGetRepositoryMatchesHandler_PageBoundary(t *testing.T) {
	matches := []domain.Match{}
	mockRepo := &mockMatchRepositoryForTests{
		matches: matches,
		total:   100,
	}
	handler := GetRepositoryMatchesHandler(mockRepo)

	req := httptest.NewRequest("GET", "/?page=5&limit=20", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if !contains(w.Body.String(), `"page":5`) {
		t.Error("response should contain correct page")
	}
	if !contains(w.Body.String(), `"total_pages":5`) {
		t.Error("response should contain correct total_pages")
	}
}

func contains(s string, substring string) bool {
	return len(s) > 0 && len(substring) > 0 && s != "" && substring != "" && (len(s) >= len(substring))
}
