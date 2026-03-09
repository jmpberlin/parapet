package crawler

import (
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

const (
	bcListingURL = "https://www.bleepingcomputer.com"
)

type BCScraper struct {
	listingURL string
}

func NewBCScraper() *BCScraper {
	return &BCScraper{
		listingURL: bcListingURL,
	}
}

func (s *BCScraper) Name() string {
	return "BleepingComputer"
}

func (s *BCScraper) FetchArticles(since time.Time) ([]domain.Article, error) {
	doc, err := fetchDocument(s.listingURL)
	if err != nil {
		return nil, err
	}
	articles, extractErrs := s.extractArticleList(since, doc)

	articles, contentErrs := s.addArticleContent(articles)
	allErrs := append(extractErrs, contentErrs...)

	if allErrs.HasErrors() {
		return articles, allErrs
	}
	return articles, nil
}

func (s *BCScraper) extractArticleList(since time.Time, doc *goquery.Document) ([]domain.Article, ScraperErrors) {
	var articles []domain.Article
	var errs ScraperErrors

	doc.Find("#bc-home-news-main-wrap li").Each(func(i int, sel *goquery.Selection) {

		category := sel.Find("span.bc_news_cat a").Text()
		if category == "Deals" {
			return
		}
		headlineTag := sel.Find("h4 a")
		headline := headlineTag.Text()

		href, exists := headlineTag.Attr("href")
		if !exists || !strings.HasPrefix(href, "https") {
			return
		}

		listItems := sel.Find("ul li")
		dateStr := strings.TrimSpace(listItems.Eq(1).Text())
		timeStr := strings.TrimSpace(listItems.Eq(2).Text())

		publishedAt, err := time.Parse("January 02, 2006 03:04 PM", dateStr+" "+timeStr)
		if err != nil {
			errs = append(errs, &ScraperError{Scraper: s.Name(), Err: err})
			return
		}

		if publishedAt.Before(since) {
			return
		}

		articles = append(articles, domain.Article{
			Headline:    headline,
			SourceURL:   href,
			HostDomain:  domain.BleepingComputer,
			PublishedAt: publishedAt,
			CrawledAt:   time.Now(),
		})
	})

	return articles, errs
}

func (s *BCScraper) addArticleContent(articles []domain.Article) ([]domain.Article, ScraperErrors) {
	var errs ScraperErrors
	for i := range articles {
		// avoid throttling or blocking by bleeping computer
		time.Sleep(750 * time.Millisecond)
		doc, err := fetchDocument(articles[i].SourceURL)

		if err != nil {
			errs = append(errs, &ScraperError{Scraper: s.Name(), Err: err})
			continue
		}
		var contentBuilder strings.Builder

		articleBody := doc.Find("div.articleBody")
		contentHTML, err := articleBody.Html()
		if err != nil {
			errs = append(errs, &ScraperError{Scraper: s.Name(), Err: err})
			continue
		}
		articles[i].ContentHTML = contentHTML

		articleBody.Find("p").Each(func(i int, sel *goquery.Selection) {
			text := strings.TrimSpace(sel.Text())
			if text != "" {
				contentBuilder.WriteString(text)
				contentBuilder.WriteString("\n\n")
			}
		})
		content := strings.TrimSpace(contentBuilder.String())
		articles[i].ContentCleaned = content
	}

	return articles, nil
}
