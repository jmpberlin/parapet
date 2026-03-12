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

type CrawlerOrchestrator struct {
	scrapers []SourceScraper
}

func NewCrawlerOrchestrator(scrapers []SourceScraper) *CrawlerOrchestrator {
	return &CrawlerOrchestrator{
		scrapers: scrapers,
	}
}

func (o *CrawlerOrchestrator) FetchArticles(since time.Time) ([]domain.Article, []error) {
	var allArticles []domain.Article
	var scraperErrs ScraperErrors

	for _, scraper := range o.scrapers {
		scrapedArticles, err := scraper.FetchArticles(since)
		if err != nil {
			scraperErrs = append(scraperErrs, &ScraperError{Scraper: scraper.Name(), Err: err})
		}
		allArticles = append(allArticles, scrapedArticles...)
	}
	if !scraperErrs.HasErrors() {
		return allArticles, nil
	}

	errs := make([]error, len(scraperErrs))
	for i, e := range scraperErrs {
		errs[i] = e
	}
	return allArticles, errs
}
