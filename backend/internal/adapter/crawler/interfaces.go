package crawler

import (
	"time"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

type SourceScraper interface {
	FetchArticles(since time.Time) ([]domain.Article, error)
	Name() string
}
