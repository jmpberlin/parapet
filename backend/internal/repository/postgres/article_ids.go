package postgres

import (
	"fmt"

	"github.com/lib/pq"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

const maxArticleFanOut = 20

const articleColumns = "id, source_url, host_domain, headline, author, content_html, content_cleaned, published_at, crawled_at, processed_at, updated_at"

func scanArticle(row interface{ Scan(...any) error }) (domain.Article, error) {
	var a domain.Article
	err := row.Scan(&a.ID, &a.SourceURL, &a.HostDomain, &a.Headline, &a.Author,
		&a.ContentHTML, &a.ContentCleaned, &a.PublishedAt, &a.CrawledAt,
		&a.ProcessedAt, &a.UpdatedAt)
	return a, err
}

// GetByIDs returns the articles for the given IDs in a single query.
// The result is capped to keep one malformed match from triggering an
// unbounded fan-out per detail request.
func (r *ArticleRepository) GetByIDs(ids []string) ([]domain.Article, error) {
	if len(ids) == 0 {
		return []domain.Article{}, nil
	}
	if len(ids) > maxArticleFanOut {
		ids = ids[:maxArticleFanOut]
	}

	rows, err := r.db.Query(`
		SELECT `+articleColumns+`
		FROM articles WHERE id = ANY($1)
	`, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to get articles by ids: %w", err)
	}
	defer rows.Close()

	byID := make(map[string]domain.Article, len(ids))
	for rows.Next() {
		a, err := scanArticle(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}
		byID[a.ID] = a
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate articles: %w", err)
	}

	ordered := make([]domain.Article, 0, len(ids))
	for _, id := range ids {
		if a, ok := byID[id]; ok {
			ordered = append(ordered, a)
		}
	}
	return ordered, nil
}
