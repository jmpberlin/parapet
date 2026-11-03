package postgres_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmpberlin/nightwatch/backend/internal/domain"
	postgres "github.com/jmpberlin/nightwatch/backend/internal/repository/postgres"
)

func truncateArticles(t *testing.T) {
	t.Helper()
	_, err := testDB.Exec(`TRUNCATE TABLE articles`)
	if err != nil {
		t.Fatalf("failed to truncate articles: %s", err)
	}
}

func newTestArticle(daysAgo int) domain.Article {
	published := time.Now().Add(-time.Duration(daysAgo) * 24 * time.Hour)
	return domain.Article{
		ID:          uuid.New().String(),
		SourceURL:   fmt.Sprintf("https://example.com/%s", uuid.New().String()),
		HostDomain:  domain.BleepingComputer,
		Headline:    fmt.Sprintf("Article from %d days ago", daysAgo),
		Author:      "Test Author",
		PublishedAt: published,
		CrawledAt:   time.Now(),
	}
}

func TestArticleRepository_GetByDaysPaginated_FirstPage(t *testing.T) {
	truncateArticles(t)
	repo := postgres.NewArticleRepository(testDB)

	// distinct published_at values, all within the 7-day window
	for i := 0; i < 25; i++ {
		a := newTestArticle(0)
		a.PublishedAt = time.Now().Add(-time.Duration(i) * time.Minute)
		if err := repo.Save(a); err != nil {
			t.Fatalf("failed to save article %d: %s", i, err)
		}
	}

	articles, total, err := repo.GetByDaysPaginated(7, 1, 10)
	if err != nil {
		t.Fatalf("failed to get paginated articles: %s", err)
	}
	if len(articles) != 10 {
		t.Errorf("expected 10 articles on page 1, got %d", len(articles))
	}
	if total != 25 {
		t.Errorf("expected total 25, got %d", total)
	}
}

func TestArticleRepository_GetByDaysPaginated_SecondPage(t *testing.T) {
	truncateArticles(t)
	repo := postgres.NewArticleRepository(testDB)

	for i := 0; i < 25; i++ {
		a := newTestArticle(0)
		a.PublishedAt = time.Now().Add(-time.Duration(i) * time.Minute)
		if err := repo.Save(a); err != nil {
			t.Fatalf("failed to save article %d: %s", i, err)
		}
	}

	first, _, err := repo.GetByDaysPaginated(7, 1, 10)
	if err != nil {
		t.Fatalf("failed to get page 1: %s", err)
	}
	second, total, err := repo.GetByDaysPaginated(7, 2, 10)
	if err != nil {
		t.Fatalf("failed to get page 2: %s", err)
	}
	if len(second) != 10 {
		t.Errorf("expected 10 articles on page 2, got %d", len(second))
	}
	if total != 25 {
		t.Errorf("expected total 25, got %d", total)
	}
	if first[0].ID == second[0].ID {
		t.Error("expected page 2 to return different articles than page 1")
	}
}

func TestArticleRepository_GetByDaysPaginated_LastPagePartial(t *testing.T) {
	truncateArticles(t)
	repo := postgres.NewArticleRepository(testDB)

	for i := 0; i < 25; i++ {
		a := newTestArticle(0)
		a.PublishedAt = time.Now().Add(-time.Duration(i) * time.Minute)
		if err := repo.Save(a); err != nil {
			t.Fatalf("failed to save article %d: %s", i, err)
		}
	}

	articles, total, err := repo.GetByDaysPaginated(7, 3, 10)
	if err != nil {
		t.Fatalf("failed to get last page: %s", err)
	}
	if len(articles) != 5 {
		t.Errorf("expected 5 articles on last page, got %d", len(articles))
	}
	if total != 25 {
		t.Errorf("expected total 25, got %d", total)
	}
}

func TestArticleRepository_GetByDaysPaginated_Empty(t *testing.T) {
	truncateArticles(t)
	repo := postgres.NewArticleRepository(testDB)

	articles, total, err := repo.GetByDaysPaginated(7, 1, 10)
	if err != nil {
		t.Fatalf("failed to get paginated articles: %s", err)
	}
	if len(articles) != 0 {
		t.Errorf("expected 0 articles, got %d", len(articles))
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
}

func TestArticleRepository_GetByDaysPaginated_DaysFilter(t *testing.T) {
	truncateArticles(t)
	repo := postgres.NewArticleRepository(testDB)

	// 3 recent articles (within 7 days), 2 old articles (older than 7 days)
	for i := 0; i < 3; i++ {
		if err := repo.Save(newTestArticle(i)); err != nil {
			t.Fatalf("failed to save recent article %d: %s", i, err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := repo.Save(newTestArticle(10 + i)); err != nil {
			t.Fatalf("failed to save old article %d: %s", i, err)
		}
	}

	articles, total, err := repo.GetByDaysPaginated(7, 1, 10)
	if err != nil {
		t.Fatalf("failed to get paginated articles: %s", err)
	}
	if total != 3 {
		t.Errorf("expected total 3 within 7 days, got %d", total)
	}
	if len(articles) != 3 {
		t.Errorf("expected 3 articles on page, got %d", len(articles))
	}
}

func TestArticleRepository_GetByDaysPaginated_StableOrder(t *testing.T) {
	truncateArticles(t)
	repo := postgres.NewArticleRepository(testDB)

	// Same published_at for all: ordering must fall back to id for stability
	published := time.Now().Add(-time.Hour)
	for i := 0; i < 5; i++ {
		a := newTestArticle(0)
		a.PublishedAt = published
		if err := repo.Save(a); err != nil {
			t.Fatalf("failed to save article %d: %s", i, err)
		}
	}

	first, _, err := repo.GetByDaysPaginated(7, 1, 10)
	if err != nil {
		t.Fatalf("failed to get page 1: %s", err)
	}
	second, _, err := repo.GetByDaysPaginated(7, 1, 10)
	if err != nil {
		t.Fatalf("failed to get page 1 again: %s", err)
	}
	for i := range first {
		if first[i].ID != second[i].ID {
			t.Errorf("expected stable order, article %d differs between calls", i)
		}
	}
}
