package crawler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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

func bcListItemHTML(category, href, headline, date, timeStr string) string {
	return "<li>" +
		`<span class="bc_news_cat"><a>` + category + `</a></span>` +
		`<h4><a href="` + href + `">` + headline + `</a></h4>` +
		"<ul><li>x</li><li>" + date + "</li><li>" + timeStr + "</li></ul>" +
		"</li>"
}

func TestBCScraper_ExtractArticleList_FiltersBySince(t *testing.T) {
	html := `<!DOCTYPE html><html><body><ul id="bc-home-news-main-wrap">` +
		bcListItemHTML("Security", "https://www.bleepingcomputer.com/news/security/older", "Older", "November 05, 2026", "11:00 AM") +
		bcListItemHTML("Security", "https://www.bleepingcomputer.com/news/security/exactly-at-since", "Exactly At Since", "November 06, 2026", "12:00 PM") +
		bcListItemHTML("Security", "https://www.bleepingcomputer.com/news/security/newer", "Newer", "November 06, 2026", "01:00 PM") +
		"</ul></body></html>"

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)

	scraper := NewBCScraper()
	since := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	articles, errs := scraper.extractArticleList(since, doc)

	require.False(t, errs.HasErrors())
	require.Len(t, articles, 2)
	assert.Equal(t, "Exactly At Since", articles[0].Headline)
	assert.Equal(t, since, articles[0].PublishedAt)
	assert.Equal(t, "Newer", articles[1].Headline)
}

func TestBCScraper_ExtractArticleList_SkipsInsecureAndRelativeLinks(t *testing.T) {
	html := `<!DOCTYPE html><html><body><ul id="bc-home-news-main-wrap">` +
		bcListItemHTML("Security", "http://www.bleepingcomputer.com/news/security/insecure", "Insecure", "November 06, 2026", "09:00 AM") +
		bcListItemHTML("Security", "/news/security/relative", "Relative", "November 06, 2026", "10:00 AM") +
		bcListItemHTML("Security", "https://bleepingcomputer.com/news/security/no-www", "No WWW", "November 06, 2026", "11:00 AM") +
		bcListItemHTML("Security", "https://www.bleepingcomputer.com/news/security/ok-post", "OK", "November 06, 2026", "12:00 PM") +
		"</ul></body></html>"

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)

	articles, errs := NewBCScraper().extractArticleList(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), doc)

	require.False(t, errs.HasErrors())
	require.Len(t, articles, 1)
	assert.Equal(t, "OK", articles[0].Headline)
}

func TestBCScraper_ExtractArticleList_BadDateRecordsError(t *testing.T) {
	html := `<!DOCTYPE html><html><body><ul id="bc-home-news-main-wrap">` +
		bcListItemHTML("Security", "https://www.bleepingcomputer.com/news/security/bad-date", "Bad Date", "not a date", "no time") +
		bcListItemHTML("Security", "https://www.bleepingcomputer.com/news/security/good-date", "Good Date", "November 06, 2026", "12:00 PM") +
		"</ul></body></html>"

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)

	articles, errs := NewBCScraper().extractArticleList(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), doc)

	require.True(t, errs.HasErrors())
	assert.Contains(t, errs.Error(), "BleepingComputer")
	require.Len(t, articles, 1)
	assert.Equal(t, "Good Date", articles[0].Headline)
}

func TestBCScraper_AddArticleContent(t *testing.T) {
	t.Run("fills content from article page", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `<!DOCTYPE html><html><body><div class="articleBody"><p>First paragraph.</p><p>Second paragraph.</p></div></body></html>`)
		}))
		defer srv.Close()

		articles := []domain.Article{{Headline: "Story", SourceURL: srv.URL + "/news/security/story"}}

		got, errs := NewBCScraper().addArticleContent(articles)

		assert.False(t, errs.HasErrors())
		require.Len(t, got, 1)
		assert.Contains(t, got[0].ContentHTML, "First paragraph.")
		assert.Equal(t, "First paragraph.\n\nSecond paragraph.", got[0].ContentCleaned)
	})

	t.Run("records error for failing article page", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "server error", http.StatusInternalServerError)
		}))
		defer srv.Close()

		articles := []domain.Article{{Headline: "Story", SourceURL: srv.URL + "/news/security/story"}}

		got, errs := NewBCScraper().addArticleContent(articles)

		require.True(t, errs.HasErrors())
		assert.Len(t, got, 1)
		assert.Empty(t, got[0].ContentHTML)
	})
}
