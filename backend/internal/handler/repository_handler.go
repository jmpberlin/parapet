package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

type WatchedRepoRepository interface {
	GetAll() ([]domain.WatchedRepository, error)
	GetByID(id string) (*domain.WatchedRepository, error)
	Save(repo domain.WatchedRepository) error
}

type DepRepository interface {
	GetByRepoID(id string) ([]domain.RepositoryDependency, error)
}

type MatchRepository interface {
	GetByRepositoryID(id string) ([]domain.Match, error)
}

type repoListItem struct {
	ID             string `json:"id"`
	OwnerName      string `json:"owner_name"`
	RepositoryName string `json:"repository_name"`
	GitProvider    string `json:"git_provider"`
}

type repoDetail struct {
	ID             string                        `json:"id"`
	OwnerName      string                        `json:"owner_name"`
	RepositoryName string                        `json:"repository_name"`
	GitProvider    string                        `json:"git_provider"`
	LastFetchedAt  *time.Time                    `json:"last_fetched_at"`
	Dependencies   []domain.RepositoryDependency `json:"dependencies"`
	Matches        []domain.Match                `json:"matches"`
}

type createRepoRequest struct {
	OwnerName      string `json:"owner_name"`
	RepositoryName string `json:"repository_name"`
	GitProvider    string `json:"git_provider"`
}

func GetRepositoriesHandler(repo WatchedRepoRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repos, err := repo.GetAll()
		if err != nil {
			http.Error(w, `{"error": "failed to fetch repositories"}`, http.StatusInternalServerError)
			return
		}

		items := make([]repoListItem, len(repos))
		for i, r := range repos {
			items[i] = repoListItem{
				ID:             r.ID,
				OwnerName:      r.OwnerName,
				RepositoryName: r.RepositoryName,
				GitProvider:    string(r.GitProvider),
			}
		}
		writeJSON(w, items)
	}
}

func GetRepositoryDetailHandler(repoRepo WatchedRepoRepository, depRepo DepRepository, matchRepo MatchRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		repo, err := repoRepo.GetByID(id)
		if err != nil {
			http.Error(w, `{"error": "failed to fetch repository"}`, http.StatusInternalServerError)
			return
		}
		if repo == nil {
			http.Error(w, `{"error": "repository not found"}`, http.StatusNotFound)
			return
		}

		deps, err := depRepo.GetByRepoID(id)
		if err != nil {
			http.Error(w, `{"error": "failed to fetch dependencies"}`, http.StatusInternalServerError)
			return
		}

		matches, err := matchRepo.GetByRepositoryID(id)
		if err != nil {
			http.Error(w, `{"error": "failed to fetch matches"}`, http.StatusInternalServerError)
			return
		}

		writeJSON(w, repoDetail{
			ID:             repo.ID,
			OwnerName:      repo.OwnerName,
			RepositoryName: repo.RepositoryName,
			GitProvider:    string(repo.GitProvider),
			LastFetchedAt:  repo.LastFetchedAt,
			Dependencies:   deps,
			Matches:        matches,
		})
	}
}

func CreateRepositoryHandler(repo WatchedRepoRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createRepoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
			return
		}

		if req.OwnerName == "" || req.RepositoryName == "" || req.GitProvider == "" {
			http.Error(w, `{"error": "owner_name, repository_name and git_provider are required"}`, http.StatusBadRequest)
			return
		}

		watched := domain.WatchedRepository{
			ID:             domain.NewID(),
			OwnerName:      req.OwnerName,
			RepositoryName: req.RepositoryName,
			GitProvider:    domain.GitProvider(req.GitProvider),
			IntegratedAt:   time.Now(),
		}

		if err := repo.Save(watched); err != nil {
			http.Error(w, `{"error": "failed to save repository"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		writeJSON(w, watched)
	}
}
