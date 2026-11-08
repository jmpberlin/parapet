package crawler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

type socketTestFeedEntry struct {
	Title       string
	Link        string
	Published   time.Time
	Author      string
	ContentHTML string
}

// socketFeedXML builds an Atom document mirroring the real
// socket.dev/api/blog/feed.atom structure (jsonfeed-to-atom output).
func socketFeedXML(feedAuthor string, entries []socketTestFeedEntry) string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	sb.WriteString(`<feed xmlns="http://www.w3.org/2005/Atom">` + "\n")
	sb.WriteString("  <title>Socket</title>\n")
	fmt.Fprintf(&sb, "  <author>\n    <name>%s</name>\n  </author>\n", feedAuthor)
	for _, e := range entries {
		sb.WriteString("  <entry>\n")
		fmt.Fprintf(&sb, "    <id>%s</id>\n", e.Link)
		fmt.Fprintf(&sb, "    <title>%s</title>\n", e.Title)
		published := e.Published.UTC().Format(time.RFC3339)
		fmt.Fprintf(&sb, "    <published>%s</published>\n", published)
		fmt.Fprintf(&sb, "    <updated>%s</updated>\n", published)
		if e.Author != "" {
			fmt.Fprintf(&sb, "    <author>\n      <name>%s</name>\n    </author>\n", e.Author)
		}
		sb.WriteString("    <link rel=\"alternate\" href=\"" + e.Link + "\"/>\n")
		fmt.Fprintf(&sb, "    <content type=\"html\"><![CDATA[%s]]></content>\n", e.ContentHTML)
		sb.WriteString("  </entry>\n")
	}
	sb.WriteString("</feed>\n")
	return sb.String()
}

// socketCardHTML mirrors the real listing markup (captured 2026-11-07):
// cards are a[href^="/blog/"] elements containing an h3 and a time. All
// classes are generated utility classes, so tests only rely on structure.
func socketCardHTML(ariaLabel, href, title, timeHTML string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, `<a class="d_flex flex-d_column ai_normal h_100%% bg_bg.base c_text.primary" aria-label=%q target="_self" href=%q>`, ariaLabel, href)
	sb.WriteString(`<div class="pos_relative w_100% ov_hidden"><picture class=""></picture></div>`)
	fmt.Fprintf(&sb, `<header class="d_flex gap_0 pt_0 pr_0 pb_lg pl_md"><div class="flex_1 min-w_0 mt_md"><h3 class="textStyle_heading-base ff_body d_-webkit-box">%s</h3></div></header>`, title)
	fmt.Fprintf(&sb, `<footer class="d_flex flex-d_column gap_2px mt_auto pt_sm pr_md pb_sm pl_md"><time class="textStyle_paragraph-sm ff_body op_0.79">%s</time></footer>`, timeHTML)
	sb.WriteString("</a>")
	return sb.String()
}

// cardDateToday formats "now" the way the listing renders card dates, so
// fallback tests are stable regardless of timezone and date boundaries.
func cardDateToday() string {
	return time.Now().UTC().Format("Jan 2, 2006")
}

func socketCategoryLinkHTML(label string) string {
	return fmt.Sprintf(`<a class="nav-dropdown-row-link" aria-label=%q target="_self" href="/blog/category/security-news"><div class="d_flex flex-d_column"><span class="textStyle_paragraph-sm">%s</span></div></a>`, label, label)
}

// socketArticlePageHTML mirrors a real article page: content lives in the
// rendered <article> element, with no __NEXT_DATA__ script tag.
func socketArticlePageHTML(bodyHTML string) string {
	return `<!DOCTYPE html><html lang="en"><head><title>Socket</title></head><body>
	<a aria-label="Socket home" href="/">Socket</a>
	<article class="">
		<header><h1 class="textStyle_display-3xl ff_body">Some Title</h1>
		<time class="textStyle_mono-md ff_mono tt_uppercase">Nov 6, 2026</time></header>
		<div class="w_100%">` + bodyHTML + `</div>
	</article>
	<footer><a href="/blog">Blog</a></footer>
	</body></html>`
}

func TestSocketDevScraper_Name(t *testing.T) {
	s := NewSocketDevScraper()
	assert.Equal(t, "Socket.dev", s.Name())
}

func TestSocketDevScraper_FetchArticlesViaFeed(t *testing.T) {
	since := time.Now().Add(-24 * time.Hour)

	contentHTML := "<p>Socket detected 84 compromised packages.</p><p>Second paragraph about npm.</p>"
	feedXML := socketFeedXML("Socket", []socketTestFeedEntry{{
		Title:       "TanStack npm Packages Compromised",
		Link:        "/blog/tanstack-npm-packages-compromised?utm_medium=feed",
		Published:   time.Now().Add(-1 * time.Hour),
		Author:      "Socket Research Team",
		ContentHTML: contentHTML,
	}})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, socketFeedPath, r.URL.Path)
		w.Header().Set("Content-Type", "application/atom+xml")
		fmt.Fprint(w, feedXML)
	}))
	defer srv.Close()

	scraper := &SocketDevScraper{baseURL: srv.URL}
	articles, err := scraper.FetchArticles(since)

	require.NoError(t, err)
	require.Len(t, articles, 1)

	a := articles[0]
	assert.Equal(t, "TanStack npm Packages Compromised", a.Headline)
	assert.Equal(t, "Socket Research Team", a.Author)
	// The feed link carries utm params; entryLink must strip them.
	gotURL, err := url.Parse(a.SourceURL)
	require.NoError(t, err)
	assert.Equal(t, "/blog/tanstack-npm-packages-compromised", gotURL.Path)
	assert.Empty(t, gotURL.RawQuery)
	assert.Equal(t, domain.SocketDev, a.HostDomain)
	assert.Contains(t, a.ContentHTML, "Socket detected 84 compromised packages")
	assert.Equal(t, "Socket detected 84 compromised packages.\n\nSecond paragraph about npm.", a.ContentCleaned)
	assert.NotEmpty(t, a.ID)
	assert.False(t, a.PublishedAt.IsZero())
	assert.False(t, a.CrawledAt.IsZero())
}

func TestSocketDevScraper_FeedFiltersBySince(t *testing.T) {
	feedXML := socketFeedXML("Socket", []socketTestFeedEntry{{
		Title:       "Old Article",
		Link:        socketBaseURL + "/blog/old-article",
		Published:   time.Now().Add(-48 * time.Hour),
		Author:      "Socket",
		ContentHTML: "<p>Old content.</p>",
	}})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		fmt.Fprint(w, feedXML)
	}))
	defer srv.Close()

	scraper := &SocketDevScraper{baseURL: srv.URL}
	articles, err := scraper.FetchArticles(time.Now().Add(-24 * time.Hour))

	require.NoError(t, err)
	assert.Empty(t, articles)
}

func TestSocketDevScraper_FeedFallsBackWhenBodyIsNotAtom(t *testing.T) {
	// An SPA can serve its HTML index with a 200 for any path; the feed
	// parse must fail (root is not <feed>) so the listing fallback runs.
	listingHTML := "<!DOCTYPE html><html><body>" +
		socketCardHTML("Title", "/blog/fallback-post", "Title", cardDateToday()) +
		"</body></html>"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/api/blog/feed.atom":
			fmt.Fprint(w, "<!DOCTYPE html><html><body>SPA shell</body></html>")
		case "/blog":
			fmt.Fprint(w, listingHTML)
		case "/blog/fallback-post":
			fmt.Fprint(w, socketArticlePageHTML("<p>fetched from the listing fallback.</p>"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	scraper := &SocketDevScraper{baseURL: srv.URL}
	articles, err := scraper.FetchArticles(time.Now().Add(-24 * time.Hour))

	require.NoError(t, err)
	require.Len(t, articles, 1)
	assert.Equal(t, "fallback-post", strings.TrimPrefix(articles[0].SourceURL, srv.URL+"/blog/"))
	assert.Contains(t, articles[0].ContentCleaned, "fetched from the listing fallback.")
}

func TestSocketDevScraper_EmptyFeedFallsBackToListing(t *testing.T) {
	listingHTML := "<!DOCTYPE html><html><body>" +
		socketCardHTML("Title", "/blog/empty-feed-fallback", "Title", cardDateToday()) +
		"</body></html>"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/api/blog/feed.atom":
			fmt.Fprint(w, `<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom"><title>Socket</title></feed>`)
		case "/blog":
			fmt.Fprint(w, listingHTML)
		case "/blog/empty-feed-fallback":
			fmt.Fprint(w, socketArticlePageHTML("<p>came from the listing.</p>"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	scraper := &SocketDevScraper{baseURL: srv.URL}
	articles, err := scraper.FetchArticles(time.Now().Add(-24 * time.Hour))

	require.NoError(t, err)
	require.Len(t, articles, 1)
	assert.Equal(t, srv.URL+"/blog/empty-feed-fallback", articles[0].SourceURL)
}

func TestSocketDevScraper_EntryWithoutAuthorUsesFeedAuthor(t *testing.T) {
	feedXML := socketFeedXML("Socket Team", []socketTestFeedEntry{{
		Title:       "No Entry Author",
		Link:        "/blog/no-entry-author",
		Published:   time.Now().Add(-1 * time.Hour),
		ContentHTML: "<p>Body text.</p>",
	}})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		fmt.Fprint(w, feedXML)
	}))
	defer srv.Close()

	scraper := &SocketDevScraper{baseURL: srv.URL}
	articles, err := scraper.FetchArticles(time.Now().Add(-24 * time.Hour))

	require.NoError(t, err)
	require.Len(t, articles, 1)
	assert.Equal(t, "Socket Team", articles[0].Author)
}

func TestSocketDevScraper_MixedFeedSkipsUnusableEntries(t *testing.T) {
	// One recent entry without content and one with an unsafe link are
	// skipped; the healthy entry is still harvested and no fallback runs.
	feedXML := `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Socket</title>
  <entry><title>No Content</title><published>%s</published><link rel="alternate" href="/blog/no-content"/></entry>
  <entry><title>Unsafe Link</title><published>%s</published><link rel="alternate" href="javascript:alert(1)"/></entry>
  <entry><title>Healthy Entry</title><published>%s</published><link rel="alternate" href="/blog/healthy-entry"/><content type="html"><![CDATA[<p>Healthy body.</p>]]></content></entry>
</feed>`
	now := time.Now().UTC().Format(time.RFC3339)
	feedXML = fmt.Sprintf(feedXML, now, now, now)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, socketFeedPath, r.URL.Path)
		w.Header().Set("Content-Type", "application/atom+xml")
		fmt.Fprint(w, feedXML)
	}))
	defer srv.Close()

	scraper := &SocketDevScraper{baseURL: srv.URL}
	articles, err := scraper.FetchArticles(time.Now().Add(-24 * time.Hour))

	require.NoError(t, err)
	require.Len(t, articles, 1)
	assert.Equal(t, "Healthy Entry", articles[0].Headline)
	assert.Equal(t, srv.URL+"/blog/healthy-entry", articles[0].SourceURL)
	assert.Equal(t, "Healthy body.", articles[0].ContentCleaned)
}

func TestSocketDevScraper_EntriesWithoutUsableContentFallBackToListing(t *testing.T) {
	// A content type other than html (or missing content) makes an entry
	// unusable; if every recent entry is unusable the listing fallback runs.
	feedXML := `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Socket</title>
  <author><name>Socket</name></author>
  <entry><title>XHTML Body</title><published>%s</published><link rel="alternate" href="/blog/xhtml-body"/></entry>
  <entry><title>No Body</title><published>%s</published><link rel="alternate" href="/blog/no-body"/><content type="html"></content></entry>
</feed>`
	now := time.Now().UTC().Format(time.RFC3339)
	feedXML = fmt.Sprintf(feedXML, now, now)

	listingHTML := "<!DOCTYPE html><html><body>" +
		socketCardHTML("From Listing", "/blog/from-listing", "From Listing", cardDateToday()) +
		"</body></html>"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/api/blog/feed.atom":
			fmt.Fprint(w, feedXML)
		case "/blog":
			fmt.Fprint(w, listingHTML)
		case "/blog/from-listing":
			fmt.Fprint(w, socketArticlePageHTML("<p>harvested from the listing after unusable feed entries.</p>"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	scraper := &SocketDevScraper{baseURL: srv.URL}
	articles, err := scraper.FetchArticles(time.Now().Add(-24 * time.Hour))

	require.NoError(t, err)
	require.Len(t, articles, 1)
	assert.Equal(t, "From Listing", articles[0].Headline)
	assert.Contains(t, articles[0].ContentCleaned, "harvested from the listing")
}

func TestSocketDevScraper_EntryMissingContentFallsBackToListing(t *testing.T) {
	feedXML := socketFeedXML("Socket", []socketTestFeedEntry{{
		Title:     "No Content",
		Link:      socketBaseURL + "/blog/no-content",
		Published: time.Now().Add(-1 * time.Hour),
	}})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/api/blog/feed.atom":
			fmt.Fprint(w, feedXML)
		case "/blog":
			fmt.Fprint(w, "<!DOCTYPE html><html><body><div>Nothing here</div></body></html>")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	scraper := &SocketDevScraper{baseURL: srv.URL}
	_, err := scraper.FetchArticles(time.Now().Add(-24 * time.Hour))

	// The listing also yields nothing, so the descriptive listing error surfaces.
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no blog posts parsed from listing page")
}

func TestSocketDevScraper_FeedFallsBackToHTMLListing(t *testing.T) {
	// Card dates have day granularity, so use "today" as the card date and a
	// comfortably older cutoff — robust against timezones and date boundaries.
	since := time.Now().Add(-72 * time.Hour)
	cardDate := cardDateToday()

	listingHTML := "<!DOCTYPE html><html lang=\"en\"><head><title>Socket Blog</title></head><body>" +
		"<nav>" + socketCategoryLinkHTML("Security") + "</nav>" +
		socketCardHTML("Free Certified Patch for Critical Rails Active Storage Vulnerability",
			"/blog/rails-active-storage-patch",
			"Free Certified Patch for Critical Rails Active Storage Vulnerability",
			cardDate) +
		socketCardHTML("GitHub to Block pull_request_target Workflows by Default Starting December 2",
			"/blog/github-to-block-pull-request-target",
			"GitHub to Block pull_request_target Workflows by Default Starting December 2",
			"Nov 5, 2026") +
		`<a aria-label="Next page" href="/blog?page=2">Next</a>` +
		"</body></html>"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/api/blog/feed.atom":
			http.NotFound(w, r)
		case "/blog":
			fmt.Fprint(w, listingHTML)
		case "/blog/rails-active-storage-patch":
			fmt.Fprint(w, socketArticlePageHTML(`<p class="textStyle_body ff_body">Rails has released security updates.</p><p>Socket released a patch.</p>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	scraper := &SocketDevScraper{baseURL: srv.URL}
	articles, err := scraper.FetchArticles(since)

	require.NoError(t, err)
	require.Len(t, articles, 1)

	a := articles[0]
	assert.Equal(t, "Free Certified Patch for Critical Rails Active Storage Vulnerability", a.Headline)
	assert.Equal(t, srv.URL+"/blog/rails-active-storage-patch", a.SourceURL)
	assert.Equal(t, domain.SocketDev, a.HostDomain)
	// Author info is not easily available from the listing markup.
	assert.Empty(t, a.Author)
	assert.Contains(t, a.ContentHTML, "Rails has released security updates")
	assert.Equal(t, "Rails has released security updates.\n\nSocket released a patch.", a.ContentCleaned)
	assert.False(t, a.PublishedAt.IsZero())
	// The listing date has no time of day and is treated as start of day UTC.
	now := time.Now().UTC()
	assert.Equal(t, time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC), a.PublishedAt)
}

func TestSocketDevScraper_ListingWithoutCardsFailsWithDescriptiveError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/api/blog/feed.atom":
			http.NotFound(w, r)
		default:
			fmt.Fprint(w, "<!DOCTYPE html><html><body><div>Nothing here</div></body></html>")
		}
	}))
	defer srv.Close()

	scraper := &SocketDevScraper{baseURL: srv.URL}
	articles, err := scraper.FetchArticles(time.Now().Add(-24 * time.Hour))

	require.Error(t, err)
	assert.Nil(t, articles)
	assert.Contains(t, err.Error(), "no blog posts parsed from listing page")
	assert.Contains(t, err.Error(), "h3")
	assert.Contains(t, err.Error(), "/blog")
}

func TestParseSocketListing_DeduplicatesSlugs(t *testing.T) {
	html := "<!DOCTYPE html><html><body>" +
		socketCardHTML("Title One", "/blog/one", "Title One", "Nov 5, 2026") +
		socketCardHTML("Title One Again", "/blog/one?utm_medium=feed", "Title One Again", "Nov 5, 2026") +
		"</body></html>"

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)

	posts := parseSocketListing(doc)
	require.Len(t, posts, 1)
	assert.Equal(t, "one", posts[0].Slug)
	assert.Equal(t, "Title One", posts[0].Title)
}

func TestParseSocketListing_SkipsNonPostLinks(t *testing.T) {
	html := "<!DOCTYPE html><html><body>" +
		// Category and pagination links have no h3/time.
		socketCategoryLinkHTML("Research") +
		`<a aria-label="Page 2" href="/blog?page=2">2</a>` +
		// Real card.
		socketCardHTML("Title", "/blog/title", "Title", "Nov 5, 2026") +
		// h3 is present but the date is unparseable, so the card is skipped.
		socketCardHTML("No Date", "/blog/no-date", "No Date", "soon") +
		"</body></html>"

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)

	posts := parseSocketListing(doc)
	require.Len(t, posts, 1)
	assert.Equal(t, "title", posts[0].Slug)
}

func TestParseSocketListing_RejectsPathTraversalSlugs(t *testing.T) {
	html := "<!DOCTYPE html><html><body>" +
		socketCardHTML("Traversal", "/blog/../../admin", "Traversal", cardDateToday()) +
		socketCardHTML("Escaped", "/blog/x/../y", "Escaped", cardDateToday()) +
		socketCardHTML("Valid", "/blog/valid-post", "Valid", cardDateToday()) +
		"</body></html>"

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)

	posts := parseSocketListing(doc)
	require.Len(t, posts, 1)
	assert.Equal(t, "valid-post", posts[0].Slug)
}

func TestParseSocketCardDate(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(
		`<div><time id="with-dt" datetime="2026-11-05T14:30:00Z">Nov 5, 2026</time><time id="text-only">Nov 5, 2026</time><time id="bad">soon</time></div>`))
	require.NoError(t, err)

	t.Run("datetime attribute wins", func(t *testing.T) {
		got, ok := parseSocketCardDate(doc.Find("#with-dt"))
		require.True(t, ok)
		assert.Equal(t, time.Date(2026, 11, 5, 14, 30, 0, 0, time.UTC), got)
	})

	t.Run("text falls back to start of day UTC", func(t *testing.T) {
		got, ok := parseSocketCardDate(doc.Find("#text-only"))
		require.True(t, ok)
		assert.Equal(t, time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC), got)
	})

	t.Run("unparseable date is skipped", func(t *testing.T) {
		_, ok := parseSocketCardDate(doc.Find("#bad"))
		assert.False(t, ok)
	})
}

func TestEntryLink_PrefersAlternateAndStripsQuery(t *testing.T) {
	entry := socketFeedEntry{
		Title: "T",
		Links: []socketFeedLink{
			{Rel: "self", Href: "https://socket.dev/api/blog/feed.atom"},
			{Rel: "alternate", Href: "https://socket.dev/blog/some-post?utm_medium=feed"},
		},
	}
	got, err := entryLink(socketBaseURL, entry)
	require.NoError(t, err)
	assert.Equal(t, "https://socket.dev/blog/some-post", got)
}

func TestEntryLink_ResolvesRelativeLinksAgainstBase(t *testing.T) {
	entry := socketFeedEntry{
		Title: "T",
		Links: []socketFeedLink{
			{Rel: "alternate", Href: "/blog/some-post"},
		},
	}
	got, err := entryLink(socketBaseURL, entry)
	require.NoError(t, err)
	assert.Equal(t, "https://socket.dev/blog/some-post", got)
}

func TestEntryLink_RejectsUnsafeLinks(t *testing.T) {
	tests := []struct {
		name string
		href string
	}{
		{name: "javascript scheme", href: "javascript:alert(1)"},
		{name: "data scheme", href: "data:text/html,<script>alert(1)</script>"},
		{name: "file scheme", href: "file:///etc/passwd"},
		{name: "foreign host", href: "https://evil.example/blog/post"},
		{name: "host with different port", href: "https://socket.dev:8080/blog/post"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			entry := socketFeedEntry{
				Title: "T",
				Links: []socketFeedLink{{Rel: "alternate", Href: tc.href}},
			}
			_, err := entryLink(socketBaseURL, entry)
			require.Error(t, err)
		})
	}
}

func TestEntryLink_FallsBackToFirstLinkWithHref(t *testing.T) {
	t.Run("no alternate link", func(t *testing.T) {
		entry := socketFeedEntry{
			Title: "T",
			Links: []socketFeedLink{
				{Rel: "self", Href: "https://socket.dev/api/blog/feed.atom"},
				{Href: "https://socket.dev/blog/some-post"},
			},
		}
		got, err := entryLink(socketBaseURL, entry)
		require.NoError(t, err)
		assert.Equal(t, "https://socket.dev/blog/some-post", got)
	})

	t.Run("alternate link with empty href is skipped", func(t *testing.T) {
		entry := socketFeedEntry{
			Title: "T",
			Links: []socketFeedLink{
				{Rel: "alternate"},
				{Href: "https://socket.dev/blog/some-post"},
			},
		}
		got, err := entryLink(socketBaseURL, entry)
		require.NoError(t, err)
		assert.Equal(t, "https://socket.dev/blog/some-post", got)
	})

	t.Run("no usable link errors", func(t *testing.T) {
		entry := socketFeedEntry{Title: "T"}
		_, err := entryLink(socketBaseURL, entry)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "has no link")
	})
}

func TestFetchArticleContent_ProseFallback(t *testing.T) {
	pageHTML := `<!DOCTYPE html><html><body>
		<div class="prose rich-text"><p>Prose fallback content.</p></div>
	</body></html>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, pageHTML)
	}))
	defer srv.Close()

	content, err := (&SocketDevScraper{baseURL: srv.URL}).fetchArticleContent(srv.URL + "/blog/some-post")
	require.NoError(t, err)
	assert.Contains(t, content.html, "Prose fallback content.")
	assert.Equal(t, "Prose fallback content.", content.cleaned)
}

func TestFetchArticleContent_NoContentErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<!DOCTYPE html><html><body><div>No article here</div></body></html>")
	}))
	defer srv.Close()

	_, err := (&SocketDevScraper{baseURL: srv.URL}).fetchArticleContent(srv.URL + "/blog/some-post")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no article content found at")
}

func TestSocketDevScraper_FetchArticles_ListingFetchError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	scraper := &SocketDevScraper{baseURL: srv.URL}
	articles, err := scraper.FetchArticles(time.Now().Add(-24 * time.Hour))

	assert.Error(t, err)
	assert.Nil(t, articles)
}

func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single paragraph",
			input:    "<p>Hello world.</p>",
			expected: "Hello world.",
		},
		{
			name:     "multiple paragraphs",
			input:    "<p>First.</p><p>Second.</p>",
			expected: "First.\n\nSecond.",
		},
		{
			name:     "nested tags",
			input:    "<p>Hello <strong>bold</strong> world.</p>",
			expected: "Hello bold world.",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := stripHTMLTags(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
