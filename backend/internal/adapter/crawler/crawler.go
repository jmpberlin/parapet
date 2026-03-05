package crawler

import (
	"fmt"
	"strings"
	"time"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

type ScraperError struct {
	Scraper string
	Err     error
}

func (e *ScraperError) Error() string {
	return fmt.Sprintf("scraper %q failed: %v", e.Scraper, e.Err)
}

func (e *ScraperError) Unwrap() error {
	return e.Err
}

type ScraperErrors []*ScraperError

func (se ScraperErrors) Error() string {
	msgs := make([]string, len(se))
	for i, e := range se {
		msgs[i] = e.Error()
	}
	return strings.Join(msgs, "; ")
}

func (se ScraperErrors) HasErrors() bool {
	return len(se) > 0
}

type SourceScraper interface {
	FetchArticles(since time.Time) ([]domain.Article, error)
	Name() string
}

type CrawlerOrchestrator struct {
	scrapers       []SourceScraper
	lookbackPeriod time.Duration
}

func NewCrawlerOrchestrator(scrapers []SourceScraper, lookbackPeriod time.Duration) *CrawlerOrchestrator {
	return &CrawlerOrchestrator{
		scrapers:       scrapers,
		lookbackPeriod: lookbackPeriod,
	}
}

func (o *CrawlerOrchestrator) FetchAll() ([]domain.Article, error) {
	var allArticles []domain.Article
	var errs ScraperErrors
	since := time.Now().Add(-o.lookbackPeriod)

	for _, scraper := range o.scrapers {
		scrapedArticles, err := scraper.FetchArticles(since)
		if err != nil {
			errs = append(errs, &ScraperError{Scraper: scraper.Name(), Err: err})
		}
		allArticles = append(allArticles, scrapedArticles...)
	}
	if errs.HasErrors() {
		return allArticles, errs
	}
	return allArticles, nil
}
