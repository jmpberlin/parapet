package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

type ArticleRepository interface {
	GetByDays(days int) ([]domain.Article, error)
	GetByID(id string) (*domain.Article, error)
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

		writeJSON(w, articles)
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

		writeJSON(w, article)
	}
}
