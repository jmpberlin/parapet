package crawler

import (
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

const (
	socketBaseURL  = "https://socket.dev"
	socketFeedPath = "/api/blog/feed.atom"
)

type SocketDevScraper struct {
	baseURL string
}

func NewSocketDevScraper() *SocketDevScraper {
	return &SocketDevScraper{baseURL: socketBaseURL}
}

func (s *SocketDevScraper) Name() string {
	return "Socket.dev"
}

// Feed-first: Socket.dev publishes a full-content Atom feed for the blog,
// which is far more stable than parsing the rendered HTML.
type socketFeed struct {
	XMLName xml.Name          `xml:"feed"`
	Entries []socketFeedEntry `xml:"entry"`
	Author  *socketFeedAuthor `xml:"author"`
}

type socketFeedEntry struct {
	Title     string             `xml:"title"`
	Published time.Time          `xml:"published"`
	Links     []socketFeedLink   `xml:"link"`
	Authors   []socketFeedAuthor `xml:"author"`
	Content   *socketFeedContent `xml:"content"`
}

type socketFeedLink struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

type socketFeedAuthor struct {
	Name string `xml:"name"`
}

type socketFeedContent struct {
	Type string `xml:"type,attr"`
	HTML string `xml:",chardata"`
}

func (s *SocketDevScraper) FetchArticles(since time.Time) ([]domain.Article, error) {
	articles, err := s.fetchFeedArticles(since)
	if err == nil {
		return articles, nil
	}

	slog.Warn("socket.dev: falling back to HTML listing", "err", err)
	return s.fetchListedArticles(since)
}

func (s *SocketDevScraper) fetchFeedArticles(since time.Time) ([]domain.Article, error) {
	body, err := fetchRaw(s.baseURL + socketFeedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch blog feed: %w", err)
	}

	var feed socketFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("failed to parse blog feed: %w", err)
	}
	if len(feed.Entries) == 0 {
		return nil, fmt.Errorf("blog feed at %s contains no entries", s.baseURL+socketFeedPath)
	}

	feedAuthor := ""
	if feed.Author != nil {
		feedAuthor = feed.Author.Name
	}

	var articles []domain.Article
	var sinceFiltered, skippedHard int
	var skippedTitles []string
	for _, entry := range feed.Entries {
		if !entry.Published.After(since) {
			sinceFiltered++
			continue
		}

		link, err := entryLink(s.baseURL, entry)
		if err != nil {
			skippedHard++
			skippedTitles = append(skippedTitles, entry.Title)
			continue
		}

		content, err := entryContentHTML(entry)
		if err != nil {
			skippedHard++
			skippedTitles = append(skippedTitles, entry.Title)
			continue
		}

		author := feedAuthor
		if len(entry.Authors) > 0 {
			author = entry.Authors[0].Name
		}

		articles = append(articles, domain.Article{
			ID:             domain.NewID(),
			Headline:       entry.Title,
			Author:         author,
			SourceURL:      link,
			HostDomain:     domain.SocketDev,
			PublishedAt:    entry.Published,
			CrawledAt:      time.Now(),
			ContentHTML:    content,
			ContentCleaned: stripHTMLTags(content),
		})
	}

	// A feed that yields zero usable recent entries (rather than genuinely
	// none) means its shape has changed; error so the caller falls back to
	// the HTML listing instead of silently harvesting nothing.
	if len(articles) == 0 && len(feed.Entries) > sinceFiltered {
		return nil, fmt.Errorf("blog feed at %s has no usable recent entries (%d unusable)", s.baseURL+socketFeedPath, skippedHard)
	}
	if len(skippedTitles) > 0 {
		slog.Warn("socket.dev: skipped unusable feed entries", "titles", skippedTitles)
	}
	return articles, nil
}

// entryContentHTML extracts the HTML payload of a feed entry. Atom defaults
// to type="text" and also allows "xhtml", neither of which chardata captures
// as usable HTML, so only explicit type="html" is accepted.
func entryContentHTML(entry socketFeedEntry) (string, error) {
	c := entry.Content
	if c == nil || (c.Type != "" && c.Type != "html") || c.HTML == "" {
		return "", fmt.Errorf("feed entry %q is missing HTML content", entry.Title)
	}
	return c.HTML, nil
}

func entryLink(baseURL string, entry socketFeedEntry) (string, error) {
	alternate := ""
	other := ""
	for _, l := range entry.Links {
		if l.Href == "" {
			continue
		}
		if alternate == "" && l.Rel == "alternate" {
			alternate = l.Href
			continue
		}
		if other == "" && l.Rel != "self" {
			other = l.Href
		}
	}

	// The self link points back at the feed itself, so it is only used as
	// a last resort — never as an article URL.
	link := alternate
	if link == "" {
		link = other
	}
	if link == "" {
		return "", fmt.Errorf("feed entry %q has no link", entry.Title)
	}

	// The feed is untrusted remote input: resolve against the scraper's base
	// URL and keep only same-origin http(s) links, so SourceURL can never
	// become a javascript:/data: link or point at another host.
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid scraper base URL %q: %w", baseURL, err)
	}
	u, err := base.Parse(link)
	if err != nil {
		return "", fmt.Errorf("failed to parse feed entry link %q: %w", link, err)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return "", fmt.Errorf("feed entry %q link has unsupported scheme %q", entry.Title, u.Scheme)
	}
	if u.Host != base.Host {
		return "", fmt.Errorf("feed entry %q link points at foreign host %q", entry.Title, u.Host)
	}
	u.RawQuery = ""
	return u.String(), nil
}

type socketPost struct {
	Title       string
	Slug        string
	PublishedAt time.Time
	Author      string
}

const fetchThrottleDelay = 500 * time.Millisecond

// Fallback when the feed is unavailable: parse the rendered blog listing.
// Classes are generated utility classes, so selection relies on structure:
// post cards are a[href^="/blog/"] elements containing an h3 and a time.
func (s *SocketDevScraper) fetchListedArticles(since time.Time) ([]domain.Article, error) {
	doc, err := fetchDocument(s.baseURL + "/blog")
	if err != nil {
		return nil, err
	}

	posts := parseSocketListing(doc)
	if len(posts) == 0 {
		return nil, fmt.Errorf("no blog posts parsed from listing page %s: expected a[href^=\"/blog/\"] cards containing h3 and time elements", s.baseURL+"/blog")
	}

	var filtered []socketPost
	for _, post := range posts {
		if post.PublishedAt.After(since) {
			filtered = append(filtered, post)
		}
	}

	return s.buildArticles(filtered)
}

func parseSocketListing(doc *goquery.Document) []socketPost {
	var posts []socketPost
	seen := make(map[string]bool)

	doc.Find("a[href^='/blog/']").Each(func(_ int, sel *goquery.Selection) {
		href, _ := sel.Attr("href")
		slug := strings.TrimSuffix(strings.TrimPrefix(href, "/blog/"), "/")
		if i := strings.IndexAny(slug, "?#"); i >= 0 {
			slug = slug[:i]
		}
		// A slug containing "/" or ".." would escape the /blog/<slug> URL
		// shape that buildArticles constructs.
		if slug == "" || seen[slug] || strings.Contains(slug, "/") || strings.Contains(slug, "..") {
			return
		}

		// Cards without an h3+time are category or pagination links.
		heading := sel.Find("h3").First()
		timeEl := sel.Find("time").First()
		if heading.Length() == 0 || timeEl.Length() == 0 {
			return
		}

		title := strings.TrimSpace(heading.Text())
		if title == "" {
			title, _ = sel.Attr("aria-label")
		}

		publishedAt, ok := parseSocketCardDate(timeEl)
		if !ok {
			return
		}

		seen[slug] = true
		posts = append(posts, socketPost{
			Title:       title,
			Slug:        slug,
			PublishedAt: publishedAt,
		})
	})
	return posts
}

func parseSocketCardDate(timeEl *goquery.Selection) (time.Time, bool) {
	if dt, exists := timeEl.Attr("datetime"); exists && dt != "" {
		if t, err := time.Parse(time.RFC3339, dt); err == nil {
			return t, true
		}
	}

	// Listing dates have no time of day; treat them as start of day UTC.
	if day, err := time.Parse("Jan 2, 2006", strings.TrimSpace(timeEl.Text())); err == nil {
		return time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC), true
	}
	return time.Time{}, false
}

func (s *SocketDevScraper) buildArticles(posts []socketPost) ([]domain.Article, error) {
	var articles []domain.Article
	var errs ScraperErrors

	for i, post := range posts {
		if i > 0 {
			time.Sleep(fetchThrottleDelay)
		}
		articleURL := s.baseURL + "/blog/" + post.Slug
		content, err := s.fetchArticleContent(articleURL)
		if err != nil {
			slog.Warn("socket.dev: failed to fetch article content", "url", articleURL, "err", err)
			errs = append(errs, &ScraperError{Scraper: s.Name(), Err: err})
			continue
		}

		articles = append(articles, domain.Article{
			ID:             domain.NewID(),
			Headline:       post.Title,
			Author:         post.Author,
			SourceURL:      articleURL,
			HostDomain:     domain.SocketDev,
			PublishedAt:    post.PublishedAt,
			CrawledAt:      time.Now(),
			ContentHTML:    content.html,
			ContentCleaned: content.cleaned,
		})
	}

	if errs.HasErrors() {
		return articles, errs
	}
	return articles, nil
}

func (s *SocketDevScraper) fetchArticleContent(articleURL string) (articleContent, error) {
	doc, err := fetchDocument(articleURL)
	if err != nil {
		return articleContent{}, err
	}

	body := doc.Find("article").First()
	if body.Length() == 0 {
		body = doc.Find("[class*='prose']").First()
	}
	if body.Length() == 0 {
		return articleContent{}, fmt.Errorf("no article content found at %s", articleURL)
	}

	html, _ := body.Html()
	return articleContent{html: html, cleaned: stripHTMLTags(html)}, nil
}

type articleContent struct {
	html    string
	cleaned string
}

func stripHTMLTags(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return html
	}

	var sb strings.Builder
	doc.Find("p").Each(func(_ int, sel *goquery.Selection) {
		text := strings.TrimSpace(sel.Text())
		if text != "" {
			sb.WriteString(text)
			sb.WriteString("\n\n")
		}
	})

	result := strings.TrimSpace(sb.String())
	if result == "" {
		return strings.TrimSpace(doc.Text())
	}
	return result
}
