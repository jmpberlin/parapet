package crawler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

func socketNextDataHTML(data any) string {
	b, _ := json.Marshal(data)
	return fmt.Sprintf(`<!DOCTYPE html><html><head>
<script id="__NEXT_DATA__" type="application/json">%s</script>
</head><body><div class="prose"><p>prose content</p></div></body></html>`, string(b))
}

func socketArticleHTML(data any, proseHTML string) string {
	b, _ := json.Marshal(data)
	return fmt.Sprintf(`<!DOCTYPE html><html><head>
<script id="__NEXT_DATA__" type="application/json">%s</script>
</head><body><div class="prose">%s</div></body></html>`, string(b), proseHTML)
}

func buildBlogListing(posts []map[string]any) map[string]any {
	return map[string]any{
		"props": map[string]any{
			"pageProps": map[string]any{
				"posts": posts,
			},
		},
	}
}

// buildPortableTextBody builds a minimal Sanity Portable Text array from plain sentences.
func buildPortableTextBody(sentences ...string) []map[string]any {
	var blocks []map[string]any
	for _, s := range sentences {
		blocks = append(blocks, map[string]any{
			"_type": "block",
			"children": []map[string]any{
				{"_type": "span", "text": s},
			},
		})
	}
	return blocks
}

func buildArticlePage(proseHTML string, bodySentences ...string) map[string]any {
	return map[string]any{
		"props": map[string]any{
			"pageProps": map[string]any{
				"data": map[string]any{
					"post": map[string]any{
						"body": buildPortableTextBody(bodySentences...),
					},
				},
			},
		},
	}
}

func TestSocketDevScraper_Name(t *testing.T) {
	s := NewSocketDevScraper()
	assert.Equal(t, "Socket.dev", s.Name())
}

func TestSocketDevScraper_FetchArticles(t *testing.T) {
	since := time.Now().Add(-24 * time.Hour)
	publishedAt := time.Now().Add(-1 * time.Hour)

	listing := buildBlogListing([]map[string]any{
		{
			"_id":         "post-1",
			"title":       "TanStack npm Packages Compromised",
			"slug":        "tanstack-npm-packages-compromised",
			"description": "Socket detected 84 compromised packages.",
			"publishedAt": publishedAt.Format(time.RFC3339Nano),
			"authors":     []map[string]any{{"name": "Socket Research Team"}},
			"categories":  []map[string]any{{"title": "Research"}},
		},
	})
	articleData := buildArticlePage(
		"<p>Socket detected 84 compromised packages.</p>",
		"Socket detected 84 compromised packages.",
	)
	articleProseHTML := "<p>Socket detected 84 compromised packages.</p>"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/blog":
			fmt.Fprint(w, socketNextDataHTML(listing))
		case "/blog/tanstack-npm-packages-compromised":
			fmt.Fprint(w, socketArticleHTML(articleData, articleProseHTML))
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
	assert.Equal(t, "TanStack npm Packages Compromised", a.Headline)
	assert.Equal(t, "Socket Research Team", a.Author)
	assert.Equal(t, srv.URL+"/blog/tanstack-npm-packages-compromised", a.SourceURL)
	assert.Equal(t, domain.SocketDev, a.HostDomain)
	assert.Contains(t, a.ContentHTML, "Socket detected 84 compromised packages")
	assert.Equal(t, "Socket detected 84 compromised packages.", a.ContentCleaned)
	assert.NotEmpty(t, a.ID)
	assert.False(t, a.PublishedAt.IsZero())
	assert.False(t, a.CrawledAt.IsZero())
}

func TestSocketDevScraper_FiltersBySince(t *testing.T) {
	since := time.Now().Add(-24 * time.Hour)
	oldPost := time.Now().Add(-48 * time.Hour)

	listing := buildBlogListing([]map[string]any{
		{
			"_id":         "old-post",
			"title":       "Old Article",
			"slug":        "old-article",
			"publishedAt": oldPost.Format(time.RFC3339Nano),
			"authors":     []map[string]any{},
		},
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, socketNextDataHTML(listing))
	}))
	defer srv.Close()

	scraper := &SocketDevScraper{baseURL: srv.URL}
	articles, err := scraper.FetchArticles(since)

	require.NoError(t, err)
	assert.Empty(t, articles)
}

func TestSocketDevScraper_MultiplePostsPartialSuccess(t *testing.T) {
	since := time.Now().Add(-24 * time.Hour)
	publishedAt := time.Now().Add(-1 * time.Hour)

	listing := buildBlogListing([]map[string]any{
		{
			"_id":         "post-1",
			"title":       "Good Article",
			"slug":        "good-article",
			"publishedAt": publishedAt.Format(time.RFC3339Nano),
			"authors":     []map[string]any{{"name": "Author One"}},
		},
		{
			"_id":         "post-2",
			"title":       "Bad Article",
			"slug":        "bad-article",
			"publishedAt": publishedAt.Format(time.RFC3339Nano),
			"authors":     []map[string]any{{"name": "Author Two"}},
		},
	})
	goodData := buildArticlePage("<p>Good content.</p>", "Good content.")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/blog":
			fmt.Fprint(w, socketNextDataHTML(listing))
		case "/blog/good-article":
			fmt.Fprint(w, socketArticleHTML(goodData, "<p>Good content.</p>"))
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer srv.Close()

	scraper := &SocketDevScraper{baseURL: srv.URL}
	articles, err := scraper.FetchArticles(since)

	// Error is returned for the failed article but the successful one is still returned.
	assert.Error(t, err)
	require.Len(t, articles, 1)
	assert.Equal(t, "Good Article", articles[0].Headline)
}

func TestSocketDevScraper_NoAuthors(t *testing.T) {
	since := time.Now().Add(-24 * time.Hour)
	publishedAt := time.Now().Add(-1 * time.Hour)

	listing := buildBlogListing([]map[string]any{
		{
			"_id":         "post-1",
			"title":       "No Author Article",
			"slug":        "no-author",
			"publishedAt": publishedAt.Format(time.RFC3339Nano),
			"authors":     []map[string]any{},
		},
	})
	articleData := buildArticlePage("<p>Content.</p>", "Content.")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/blog":
			fmt.Fprint(w, socketNextDataHTML(listing))
		case "/blog/no-author":
			fmt.Fprint(w, socketArticleHTML(articleData, "<p>Content.</p>"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	scraper := &SocketDevScraper{baseURL: srv.URL}
	articles, err := scraper.FetchArticles(since)

	require.NoError(t, err)
	require.Len(t, articles, 1)
	assert.Empty(t, articles[0].Author)
}

func TestSocketDevScraper_ListingFetchError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	scraper := &SocketDevScraper{baseURL: srv.URL}
	articles, err := scraper.FetchArticles(time.Now().Add(-24 * time.Hour))

	assert.Error(t, err)
	assert.Nil(t, articles)
}

func TestPortableTextToPlain(t *testing.T) {
	blocks := buildPortableTextBody("Hello world.", "Second paragraph.")
	raw, _ := json.Marshal(blocks)
	result := portableTextToPlain(raw)
	assert.Equal(t, "Hello world.\n\nSecond paragraph.", result)
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
