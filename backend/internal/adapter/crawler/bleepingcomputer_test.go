package crawler

import (
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

func TestBCScraper_ExtractArticleList_SkipsNonArticleLinks(t *testing.T) {
	html := `<!DOCTYPE html><html><body><ul id="bc-home-news-main-wrap">
		<li>
			<span class="bc_news_cat"><a>Security</a></span>
			<h4><a href="https://evil.example/news/fake-story">Off-site Story</a></h4>
			<ul><li>x</li><li>November 06, 2026</li><li>10:00 AM</li></ul>
		</li>
		<li>
			<span class="bc_news_cat"><a>Deals</a></span>
			<h4><a href="https://www.bleepingcomputer.com/news/security/deal-post">Deal Post</a></h4>
			<ul><li>Deals</li><li>November 06, 2026</li><li>11:00 AM</li></ul>
		</li>
		<li>
			<span class="bc_news_cat"><a>Security</a></span>
			<h4><a href="https://www.bleepingcomputer.com/news/security/real-story">Real Story</a></h4>
			<ul><li>Security</li><li>November 06, 2026</li><li>12:00 PM</li></ul>
		</li>
	</ul></body></html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)

	scraper := NewBCScraper()
	articles, errs := scraper.extractArticleList(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), doc)

	// Only the same-origin, non-Deals story is harvested.
	require.Len(t, articles, 1)
	assert.Equal(t, "Real Story", articles[0].Headline)
	assert.Equal(t, "https://www.bleepingcomputer.com/news/security/real-story", articles[0].SourceURL)
	assert.Equal(t, domain.BleepingComputer, articles[0].HostDomain)
	assert.False(t, articles[0].PublishedAt.IsZero())
	assert.False(t, errs.HasErrors())
}
