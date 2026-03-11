package postgres

import (
	"database/sql"
	"fmt"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

type ArticleRepository struct {
	db *sql.DB
}

func NewArticleRepository(db *sql.DB) *ArticleRepository {
	return &ArticleRepository{db: db}
}

func (r *ArticleRepository) Save(article domain.Article) error {
	_, err := r.db.Exec(`
        INSERT INTO articles (id, source_url, host_domain, headline, author, content_html, content_cleaned, published_at, crawled_at, processed_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
        ON CONFLICT (source_url) DO UPDATE
		SET content_html = EXCLUDED.content_html,
		content_cleaned = EXCLUDED.content_cleaned,
		processed_at = EXCLUDED.processed_at,
		updated_at = NOW()
    `, article.ID, article.SourceURL, article.HostDomain, article.Headline, article.Author, article.ContentHTML, article.ContentCleaned, article.PublishedAt, article.CrawledAt, article.ProcessedAt)
	return err
}

func (r *ArticleRepository) GetByID(id string) (*domain.Article, error) {
	row := r.db.QueryRow(`
		SELECT id, source_url, host_domain, headline, author, content_html, 
		       content_cleaned, published_at, crawled_at, processed_at, updated_at
		FROM articles WHERE id = $1
	`, id)

	var a domain.Article
	err := row.Scan(&a.ID, &a.SourceURL, &a.HostDomain, &a.Headline, &a.Author,
		&a.ContentHTML, &a.ContentCleaned, &a.PublishedAt, &a.CrawledAt,
		&a.ProcessedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get article by id: %w", err)
	}
	return &a, nil
}

func (r *ArticleRepository) GetByURL(url string) (*domain.Article, error) {
	row := r.db.QueryRow(`
		SELECT id, source_url, host_domain, headline, author, content_html,
		       content_cleaned, published_at, crawled_at, processed_at, updated_at
		FROM articles WHERE source_url = $1
	`, url)

	var a domain.Article
	err := row.Scan(&a.ID, &a.SourceURL, &a.HostDomain, &a.Headline, &a.Author,
		&a.ContentHTML, &a.ContentCleaned, &a.PublishedAt, &a.CrawledAt,
		&a.ProcessedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get article by url: %w", err)
	}
	return &a, nil
}

func (r *ArticleRepository) GetUnprocessed() ([]domain.Article, error) {
	rows, err := r.db.Query(`
		SELECT id, source_url, host_domain, headline, author, content_html,
		       content_cleaned, published_at, crawled_at, processed_at, updated_at
		FROM articles WHERE processed_at IS NULL
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get unprocessed articles: %w", err)
	}
	defer rows.Close()

	var articles []domain.Article
	for rows.Next() {
		var a domain.Article
		err := rows.Scan(&a.ID, &a.SourceURL, &a.HostDomain, &a.Headline, &a.Author,
			&a.ContentHTML, &a.ContentCleaned, &a.PublishedAt, &a.CrawledAt,
			&a.ProcessedAt, &a.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}
		articles = append(articles, a)
	}
	return articles, nil
}
