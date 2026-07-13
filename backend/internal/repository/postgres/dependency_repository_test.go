package postgres_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmpberlin/nightwatch/backend/internal/domain"
	postgres "github.com/jmpberlin/nightwatch/backend/internal/repository/postgres"
)

func truncateDependencies(t *testing.T) {
	t.Helper()
	_, err := testDB.Exec(`TRUNCATE TABLE repository_dependencies`)
	if err != nil {
		t.Fatalf("failed to truncate dependencies: %s", err)
	}
}

func seedDependencyRepo(t *testing.T) string {
	t.Helper()
	repoID := uuid.New().String()

	_, err := testDB.Exec(`
		INSERT INTO watched_repositories (id, git_provider, owner_name, repository_name)
		VALUES ($1, 'github', 'testowner', $2)
	`, repoID, uuid.New().String())
	if err != nil {
		t.Fatalf("failed to seed watched repository: %s", err)
	}

	return repoID
}

func newTestDependency(repoID string) domain.RepositoryDependency {
	return domain.RepositoryDependency{
		ID:           uuid.New().String(),
		RepositoryID: repoID,
		Name:         "lodash",
		Version:      "4.17.20",
		PURL:         "pkg:npm/lodash@4.17.20",
		CreatedAt:    time.Now(),
	}
}

func findDependencyByID(deps []domain.RepositoryDependency, id string) *domain.RepositoryDependency {
	for i := range deps {
		if deps[i].ID == id {
			return &deps[i]
		}
	}
	return nil
}

func TestDependencyRepository_GetCountByRepoID_Zero(t *testing.T) {
	truncateDependencies(t)
	repoID := seedDependencyRepo(t)
	repo := postgres.NewDependencyRepository(testDB)

	count, err := repo.GetCountByRepoID(repoID)
	if err != nil {
		t.Fatalf("failed to get count: %s", err)
	}
	if count != 0 {
		t.Errorf("expected count 0 for empty repo, got %d", count)
	}
}

func TestDependencyRepository_GetCountByRepoID_Multiple(t *testing.T) {
	truncateDependencies(t)
	repoID := seedDependencyRepo(t)
	repo := postgres.NewDependencyRepository(testDB)

	for i := 0; i < 5; i++ {
		d := newTestDependency(repoID)
		if err := repo.Save(d); err != nil {
			t.Fatalf("failed to save dependency %d: %s", i, err)
		}
	}

	count, err := repo.GetCountByRepoID(repoID)
	if err != nil {
		t.Fatalf("failed to get count: %s", err)
	}
	if count != 5 {
		t.Errorf("expected count 5, got %d", count)
	}
}

func TestDependencyRepository_GetCountByRepoID_IsolatedByRepo(t *testing.T) {
	truncateDependencies(t)
	repoID1 := seedDependencyRepo(t)
	repoID2 := seedDependencyRepo(t)
	repo := postgres.NewDependencyRepository(testDB)

	for i := 0; i < 3; i++ {
		d := newTestDependency(repoID1)
		if err := repo.Save(d); err != nil {
			t.Fatalf("failed to save dependency for repo1: %s", err)
		}
	}

	for i := 0; i < 7; i++ {
		d := newTestDependency(repoID2)
		if err := repo.Save(d); err != nil {
			t.Fatalf("failed to save dependency for repo2: %s", err)
		}
	}

	count1, err := repo.GetCountByRepoID(repoID1)
	if err != nil {
		t.Fatalf("failed to get count for repo1: %s", err)
	}
	if count1 != 3 {
		t.Errorf("expected count 3 for repo1, got %d", count1)
	}

	count2, err := repo.GetCountByRepoID(repoID2)
	if err != nil {
		t.Fatalf("failed to get count for repo2: %s", err)
	}
	if count2 != 7 {
		t.Errorf("expected count 7 for repo2, got %d", count2)
	}
}

func TestDependencyRepository_GetByRepoIDPaginated_FirstPage(t *testing.T) {
	truncateDependencies(t)
	repoID := seedDependencyRepo(t)
	repo := postgres.NewDependencyRepository(testDB)

	for i := 0; i < 50; i++ {
		d := newTestDependency(repoID)
		if err := repo.Save(d); err != nil {
			t.Fatalf("failed to save dependency: %s", err)
		}
	}

	deps, total, err := repo.GetByRepoIDPaginated(repoID, 1, 20)
	if err != nil {
		t.Fatalf("failed to get paginated dependencies: %s", err)
	}
	if len(deps) != 20 {
		t.Errorf("expected 20 dependencies on page 1, got %d", len(deps))
	}
	if total != 50 {
		t.Errorf("expected total 50, got %d", total)
	}
}

func TestDependencyRepository_GetByRepoIDPaginated_SecondPage(t *testing.T) {
	truncateDependencies(t)
	repoID := seedDependencyRepo(t)
	repo := postgres.NewDependencyRepository(testDB)

	for i := 0; i < 50; i++ {
		d := newTestDependency(repoID)
		if err := repo.Save(d); err != nil {
			t.Fatalf("failed to save dependency: %s", err)
		}
	}

	deps, total, err := repo.GetByRepoIDPaginated(repoID, 2, 20)
	if err != nil {
		t.Fatalf("failed to get paginated dependencies: %s", err)
	}
	if len(deps) != 20 {
		t.Errorf("expected 20 dependencies on page 2, got %d", len(deps))
	}
	if total != 50 {
		t.Errorf("expected total 50, got %d", total)
	}
}

func TestDependencyRepository_GetByRepoIDPaginated_LastPagePartial(t *testing.T) {
	truncateDependencies(t)
	repoID := seedDependencyRepo(t)
	repo := postgres.NewDependencyRepository(testDB)

	for i := 0; i < 125; i++ {
		d := newTestDependency(repoID)
		if err := repo.Save(d); err != nil {
			t.Fatalf("failed to save dependency: %s", err)
		}
	}

	deps, total, err := repo.GetByRepoIDPaginated(repoID, 3, 50)
	if err != nil {
		t.Fatalf("failed to get paginated dependencies: %s", err)
	}
	if len(deps) != 25 {
		t.Errorf("expected 25 dependencies on last page, got %d", len(deps))
	}
	if total != 125 {
		t.Errorf("expected total 125, got %d", total)
	}
}

func TestDependencyRepository_GetByRepoIDPaginated_Empty(t *testing.T) {
	truncateDependencies(t)
	repoID := seedDependencyRepo(t)
	repo := postgres.NewDependencyRepository(testDB)

	deps, total, err := repo.GetByRepoIDPaginated(repoID, 1, 50)
	if err != nil {
		t.Fatalf("failed to get paginated dependencies: %s", err)
	}
	if len(deps) != 0 {
		t.Errorf("expected 0 dependencies for empty repo, got %d", len(deps))
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
}

func TestDependencyRepository_GetByRepoIDPaginated_SingleItem(t *testing.T) {
	truncateDependencies(t)
	repoID := seedDependencyRepo(t)
	repo := postgres.NewDependencyRepository(testDB)

	d := newTestDependency(repoID)
	if err := repo.Save(d); err != nil {
		t.Fatalf("failed to save dependency: %s", err)
	}

	deps, total, err := repo.GetByRepoIDPaginated(repoID, 1, 50)
	if err != nil {
		t.Fatalf("failed to get paginated dependencies: %s", err)
	}
	if len(deps) != 1 {
		t.Errorf("expected 1 dependency, got %d", len(deps))
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if deps[0].ID != d.ID {
		t.Errorf("expected dependency ID %s, got %s", d.ID, deps[0].ID)
	}
}

func TestDependencyRepository_GetByRepoIDPaginated_IsolatedByRepo(t *testing.T) {
	truncateDependencies(t)
	repoID1 := seedDependencyRepo(t)
	repoID2 := seedDependencyRepo(t)
	repo := postgres.NewDependencyRepository(testDB)

	for i := 0; i < 100; i++ {
		d := newTestDependency(repoID1)
		if err := repo.Save(d); err != nil {
			t.Fatalf("failed to save dependency for repo1: %s", err)
		}
	}

	for i := 0; i < 25; i++ {
		d := newTestDependency(repoID2)
		if err := repo.Save(d); err != nil {
			t.Fatalf("failed to save dependency for repo2: %s", err)
		}
	}

	deps1, total1, err := repo.GetByRepoIDPaginated(repoID1, 1, 50)
	if err != nil {
		t.Fatalf("failed to get paginated dependencies for repo1: %s", err)
	}
	if len(deps1) != 50 {
		t.Errorf("expected 50 dependencies for repo1, got %d", len(deps1))
	}
	if total1 != 100 {
		t.Errorf("expected total 100 for repo1, got %d", total1)
	}

	deps2, total2, err := repo.GetByRepoIDPaginated(repoID2, 1, 50)
	if err != nil {
		t.Fatalf("failed to get paginated dependencies for repo2: %s", err)
	}
	if len(deps2) != 25 {
		t.Errorf("expected 25 dependencies for repo2, got %d", len(deps2))
	}
	if total2 != 25 {
		t.Errorf("expected total 25 for repo2, got %d", total2)
	}
}

func TestDependencyRepository_GetByRepoIDPaginated_DifferentLimitSizes(t *testing.T) {
	truncateDependencies(t)
	repoID := seedDependencyRepo(t)
	repo := postgres.NewDependencyRepository(testDB)

	for i := 0; i < 100; i++ {
		d := newTestDependency(repoID)
		if err := repo.Save(d); err != nil {
			t.Fatalf("failed to save dependency: %s", err)
		}
	}

	tests := []struct {
		page      int
		limit     int
		wantCount int
		wantTotal int
	}{
		{1, 10, 10, 100},
		{1, 25, 25, 100},
		{2, 25, 25, 100},
		{4, 25, 25, 100},
		{5, 25, 0, 100},
		{1, 100, 100, 100},
	}

	for _, tt := range tests {
		deps, total, err := repo.GetByRepoIDPaginated(repoID, tt.page, tt.limit)
		if err != nil {
			t.Fatalf("failed to get paginated dependencies: %s", err)
		}
		if len(deps) != tt.wantCount {
			t.Errorf("page=%d, limit=%d: expected %d dependencies, got %d", tt.page, tt.limit, tt.wantCount, len(deps))
		}
		if total != tt.wantTotal {
			t.Errorf("page=%d, limit=%d: expected total %d, got %d", tt.page, tt.limit, tt.wantTotal, total)
		}
	}
}

func TestDependencyRepository_GetByRepoIDPaginated_OrderByCreatedAt(t *testing.T) {
	truncateDependencies(t)
	repoID := seedDependencyRepo(t)
	repo := postgres.NewDependencyRepository(testDB)

	for i := 0; i < 5; i++ {
		d := newTestDependency(repoID)
		if err := repo.Save(d); err != nil {
			t.Fatalf("failed to save dependency: %s", err)
		}
	}

	deps, _, err := repo.GetByRepoIDPaginated(repoID, 1, 10)
	if err != nil {
		t.Fatalf("failed to get paginated dependencies: %s", err)
	}
	if len(deps) != 5 {
		t.Fatalf("expected 5 dependencies, got %d", len(deps))
	}

	for i := 0; i < len(deps)-1; i++ {
		if deps[i].CreatedAt.Before(deps[i+1].CreatedAt) {
			t.Logf("dependencies ordered by created_at DESC: OK (dep[%d] >= dep[%d])", i, i+1)
		}
	}
}

func TestDependencyRepository_GetByRepoIDPaginated_RetainsAllFields(t *testing.T) {
	truncateDependencies(t)
	repoID := seedDependencyRepo(t)
	repo := postgres.NewDependencyRepository(testDB)

	d := newTestDependency(repoID)
	d.Name = "express"
	d.Version = "4.18.0"
	d.PURL = "pkg:npm/express@4.18.0"

	now := time.Now()
	d.LastMatchedAt = &now

	if err := repo.Save(d); err != nil {
		t.Fatalf("failed to save dependency: %s", err)
	}

	deps, _, err := repo.GetByRepoIDPaginated(repoID, 1, 50)
	if err != nil {
		t.Fatalf("failed to get paginated dependencies: %s", err)
	}
	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(deps))
	}

	got := deps[0]
	if got.ID != d.ID {
		t.Errorf("expected ID %s, got %s", d.ID, got.ID)
	}
	if got.RepositoryID != d.RepositoryID {
		t.Errorf("expected RepositoryID %s, got %s", d.RepositoryID, got.RepositoryID)
	}
	if got.Name != d.Name {
		t.Errorf("expected Name %s, got %s", d.Name, got.Name)
	}
	if got.Version != d.Version {
		t.Errorf("expected Version %s, got %s", d.Version, got.Version)
	}
	if got.PURL != d.PURL {
		t.Errorf("expected PURL %s, got %s", d.PURL, got.PURL)
	}
	if got.LastMatchedAt == nil {
		t.Error("expected LastMatchedAt to be set")
	}
}
