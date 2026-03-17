package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmpberlin/nightwatch/backend/internal/domain"
	"github.com/jmpberlin/nightwatch/backend/internal/repository/postgres"
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

type repositoryDependencyResponse struct {
	ID            string     `json:"id"`
	RepositoryID  string     `json:"repository_id"`
	Name          string     `json:"name"`
	Version       string     `json:"version"`
	PURL          string     `json:"purl"`
	CreatedAt     time.Time  `json:"created_at"`
	LastMatchedAt *time.Time `json:"last_matched_at"`
}

type matchResponse struct {
	ID               string     `json:"id"`
	VulnerabilityID  string     `json:"vulnerability_id"`
	RepositoryID     string     `json:"repository_id"`
	ComponentPURL    string     `json:"component_purl"`
	MatchedComponent string     `json:"matched_component"`
	MatchedVersion   string     `json:"matched_version"`
	Status           string     `json:"status"`
	ResolvedAt       *time.Time `json:"resolved_at"`
	CreatedAt        time.Time  `json:"created_at"`
}

type repoListItem struct {
	ID             string `json:"id"`
	OwnerName      string `json:"owner_name"`
	RepositoryName string `json:"repository_name"`
	GitProvider    string `json:"git_provider"`
}

type repoDetail struct {
	ID             string                         `json:"id"`
	OwnerName      string                         `json:"owner_name"`
	RepositoryName string                         `json:"repository_name"`
	GitProvider    string                         `json:"git_provider"`
	LastFetchedAt  *time.Time                     `json:"last_fetched_at"`
	Dependencies   []repositoryDependencyResponse `json:"dependencies"`
	Matches        []matchResponse                `json:"matches"`
}

type createRepoRequest struct {
	OwnerName      string `json:"owner_name"`
	RepositoryName string `json:"repository_name"`
	GitProvider    string `json:"git_provider"`
}

func toRepositoryDependencyResponse(d domain.RepositoryDependency) repositoryDependencyResponse {
	return repositoryDependencyResponse{
		ID:            d.ID,
		RepositoryID:  d.RepositoryID,
		Name:          d.Name,
		Version:       d.Version,
		PURL:          d.PURL,
		CreatedAt:     d.CreatedAt,
		LastMatchedAt: d.LastMatchedAt,
	}
}

func toMatchResponse(m domain.Match) matchResponse {
	return matchResponse{
		ID:               m.ID,
		VulnerabilityID:  m.VulnerabilityID,
		RepositoryID:     m.RepositoryID,
		ComponentPURL:    m.ComponentPURL,
		MatchedComponent: m.MatchedComponent,
		MatchedVersion:   m.MatchedVersion,
		Status:           string(m.Status),
		ResolvedAt:       m.ResolvedAt,
		CreatedAt:        m.CreatedAt,
	}
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

		depsResponse := make([]repositoryDependencyResponse, len(deps))
		for i, d := range deps {
			depsResponse[i] = toRepositoryDependencyResponse(d)
		}

		matchesResponse := make([]matchResponse, len(matches))
		for i, m := range matches {
			matchesResponse[i] = toMatchResponse(m)
		}

		writeJSON(w, repoDetail{
			ID:             repo.ID,
			OwnerName:      repo.OwnerName,
			RepositoryName: repo.RepositoryName,
			GitProvider:    string(repo.GitProvider),
			LastFetchedAt:  repo.LastFetchedAt,
			Dependencies:   depsResponse,
			Matches:        matchesResponse,
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
			if errors.Is(err, postgres.ErrRepositoryAlreadyExists) {
				http.Error(w, `{"error": "repository already watched"}`, http.StatusConflict)
				return
			}
			http.Error(w, `{"error": "failed to save repository"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		writeJSON(w, repoDetail{
			ID:             watched.ID,
			OwnerName:      watched.OwnerName,
			RepositoryName: watched.RepositoryName,
			GitProvider:    string(watched.GitProvider),
			LastFetchedAt:  watched.LastFetchedAt,
			Dependencies:   []repositoryDependencyResponse{},
			Matches:        []matchResponse{},
		})
	}
}
