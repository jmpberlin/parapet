package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

type ArticleRepository interface {
	GetByDays(days int) ([]domain.Article, error)
	GetByID(id string) (*domain.Article, error)
}

type articleResponse struct {
	ID             string     `json:"id"`
	SourceURL      string     `json:"source_url"`
	HostDomain     string     `json:"host_domain"`
	PublishedAt    time.Time  `json:"published_at"`
	Headline       string     `json:"headline"`
	Author         string     `json:"author"`
	ContentHTML    string     `json:"content_html"`
	ContentCleaned string     `json:"content_cleaned"`
	CrawledAt      time.Time  `json:"crawled_at"`
	ProcessedAt    *time.Time `json:"processed_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
}

func toArticleResponse(a domain.Article) articleResponse {
	return articleResponse{
		ID:             a.ID,
		SourceURL:      a.SourceURL,
		HostDomain:     string(a.HostDomain),
		PublishedAt:    a.PublishedAt,
		Headline:       a.Headline,
		Author:         a.Author,
		ContentHTML:    a.ContentHTML,
		ContentCleaned: a.ContentCleaned,
		CrawledAt:      a.CrawledAt,
		ProcessedAt:    a.ProcessedAt,
		UpdatedAt:      a.UpdatedAt,
	}
}

func GetArticlesHandler(repo ArticleRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		daysStr := r.URL.Query().Get("days")
		days := 7
		if daysStr != "" {
			parsed, err := strconv.Atoi(daysStr)
			if err != nil || parsed < 1 {
				http.Error(w, `{"error": "invalid days parameter"}`, http.StatusBadRequest)
				return
			}
			days = parsed
		}

		articles, err := repo.GetByDays(days)
		if err != nil {
			http.Error(w, `{"error": "failed to fetch articles"}`, http.StatusInternalServerError)
			return
		}

		response := make([]articleResponse, len(articles))
		for i, a := range articles {
			response[i] = toArticleResponse(a)
		}
		writeJSON(w, response)
	}
}

func GetArticleByIDHandler(repo ArticleRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		article, err := repo.GetByID(id)
		if err != nil {
			http.Error(w, `{"error": "failed to fetch article"}`, http.StatusInternalServerError)
			return
		}
		if article == nil {
			http.Error(w, `{"error": "article not found"}`, http.StatusNotFound)
			return
		}

		writeJSON(w, toArticleResponse(*article))
	}
}
