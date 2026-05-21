package crawler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

const socketBaseURL = "https://socket.dev"

type SocketDevScraper struct {
	baseURL string
}

func NewSocketDevScraper() *SocketDevScraper {
	return &SocketDevScraper{baseURL: socketBaseURL}
}

func (s *SocketDevScraper) Name() string {
	return "Socket.dev"
}

type socketListingNextData struct {
	Props struct {
		PageProps struct {
			Posts []socketPost `json:"posts"`
		} `json:"pageProps"`
	} `json:"props"`
}

type socketPost struct {
	ID          string           `json:"_id"`
	Title       string           `json:"title"`
	Slug        string           `json:"slug"`
	Description string           `json:"description"`
	PublishedAt time.Time        `json:"publishedAt"`
	Authors     []socketAuthor   `json:"authors"`
	Categories  []socketCategory `json:"categories"`
}

type socketAuthor struct {
	Name string `json:"name"`
}

type socketCategory struct {
	Title string `json:"title"`
}

// Article page: props.pageProps.data.post.body (Sanity Portable Text)
type socketArticleNextData struct {
	Props struct {
		PageProps struct {
			Data struct {
				Post struct {
					Body json.RawMessage `json:"body"`
				} `json:"post"`
			} `json:"data"`
		} `json:"pageProps"`
	} `json:"props"`
}

type ptBlock struct {
	Type     string            `json:"_type"`
	Children []json.RawMessage `json:"children"`
}

type ptSpan struct {
	Type string `json:"_type"`
	Text string `json:"text"`
}

func portableTextToPlain(body json.RawMessage) string {
	var blocks []ptBlock
	if err := json.Unmarshal(body, &blocks); err != nil {
		return ""
	}
	var sb strings.Builder
	for _, block := range blocks {
		if block.Type != "block" {
			continue
		}
		for _, childRaw := range block.Children {
			var span ptSpan
			if err := json.Unmarshal(childRaw, &span); err != nil {
				continue
			}
			if span.Type == "span" {
				sb.WriteString(span.Text)
			}
		}
		sb.WriteString("\n\n")
	}
	return strings.TrimSpace(sb.String())
}

type articleContent struct {
	html    string
	cleaned string
}

func (s *SocketDevScraper) FetchArticles(since time.Time) ([]domain.Article, error) {
	posts, err := s.fetchPostList(since)
	if err != nil {
		return nil, err
	}

	articles, errs := s.buildArticles(posts)
	if errs.HasErrors() {
		return articles, errs
	}
	return articles, nil
}

func (s *SocketDevScraper) fetchPostList(since time.Time) ([]socketPost, error) {
	doc, err := fetchDocument(s.baseURL + "/blog")
	if err != nil {
		return nil, err
	}

	jsonStr := doc.Find("script#__NEXT_DATA__").Text()
	if jsonStr == "" {
		return nil, fmt.Errorf("__NEXT_DATA__ script tag not found on blog listing page")
	}

	var nextData socketListingNextData
	if err := json.Unmarshal([]byte(jsonStr), &nextData); err != nil {
		return nil, fmt.Errorf("failed to parse blog listing __NEXT_DATA__: %w", err)
	}

	var filtered []socketPost
	for _, post := range nextData.Props.PageProps.Posts {
		if post.PublishedAt.After(since) {
			filtered = append(filtered, post)
		}
	}
	return filtered, nil
}

func (s *SocketDevScraper) buildArticles(posts []socketPost) ([]domain.Article, ScraperErrors) {
	var articles []domain.Article
	var errs ScraperErrors

	for _, post := range posts {
		articleURL := s.baseURL + "/blog/" + post.Slug
		content, err := s.fetchArticleContent(articleURL)
		if err != nil {
			slog.Warn("socket.dev: failed to fetch article content", "url", articleURL, "err", err)
			errs = append(errs, &ScraperError{Scraper: s.Name(), Err: err})
			continue
		}

		var author string
		if len(post.Authors) > 0 {
			author = post.Authors[0].Name
		}

		articles = append(articles, domain.Article{
			ID:             domain.NewID(),
			Headline:       post.Title,
			Author:         author,
			SourceURL:      articleURL,
			HostDomain:     domain.SocketDev,
			PublishedAt:    post.PublishedAt,
			CrawledAt:      time.Now(),
			ContentHTML:    content.html,
			ContentCleaned: content.cleaned,
		})
	}

	return articles, errs
}

func (s *SocketDevScraper) fetchArticleContent(url string) (articleContent, error) {
	doc, err := fetchDocument(url)
	if err != nil {
		return articleContent{}, err
	}

	// ContentCleaned: walk Portable Text body from __NEXT_DATA__
	var cleaned string
	jsonStr := doc.Find("script#__NEXT_DATA__").Text()
	if jsonStr != "" {
		var nextData socketArticleNextData
		if err := json.Unmarshal([]byte(jsonStr), &nextData); err == nil {
			cleaned = portableTextToPlain(nextData.Props.PageProps.Data.Post.Body)
		}
	}

	// ContentHTML: rendered prose element on the page
	var html string
	proseEl := doc.Find("[class*='prose']").First()
	if proseEl.Length() > 0 {
		html, _ = proseEl.Html()
	}

	if html == "" && cleaned == "" {
		return articleContent{}, fmt.Errorf("no article content found at %s", url)
	}
	if cleaned == "" {
		cleaned = stripHTMLTags(html)
	}

	return articleContent{html: html, cleaned: cleaned}, nil
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
