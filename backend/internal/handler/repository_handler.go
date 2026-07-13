package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
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
	GetByRepoIDPaginated(repoID string, page, limit int) ([]domain.RepositoryDependency, int, error)
}

type MatchRepository interface {
	GetByRepositoryID(id string) ([]domain.Match, error)
	GetByRepositoryIDPaginated(repoID string, page, limit int) ([]domain.Match, int, error)
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
	ID                 string     `json:"id"`
	OwnerName          string     `json:"owner_name"`
	RepositoryName     string     `json:"repository_name"`
	GitProvider        string     `json:"git_provider"`
	LastFetchedAt      *time.Time `json:"last_fetched_at"`
	DependencyCount    int        `json:"dependency_count"`
	MatchCount         int        `json:"match_count"`
}

type paginatedDependenciesResponse struct {
	Items      []repositoryDependencyResponse `json:"items"`
	Total      int                            `json:"total"`
	Page       int                            `json:"page"`
	Limit      int                            `json:"limit"`
	TotalPages int                            `json:"total_pages"`
}

type paginatedMatchesResponse struct {
	Items      []matchResponse `json:"items"`
	Total      int             `json:"total"`
	Page       int             `json:"page"`
	Limit      int             `json:"limit"`
	TotalPages int             `json:"total_pages"`
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

func getPaginationParams(r *http.Request, defaultLimit int) (page, limit int, err error) {
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	if pageStr == "" {
		pageStr = "1"
	}
	if limitStr == "" {
		limitStr = strconv.Itoa(defaultLimit)
	}

	page, err = strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		return 0, 0, errors.New("invalid page parameter")
	}

	limit, err = strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		return 0, 0, errors.New("invalid limit parameter")
	}

	return page, limit, nil
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

		_, depCount, err := depRepo.GetByRepoIDPaginated(id, 1, 1)
		if err != nil {
			http.Error(w, `{"error": "failed to fetch dependencies"}`, http.StatusInternalServerError)
			return
		}

		_, matchCount, err := matchRepo.GetByRepositoryIDPaginated(id, 1, 1)
		if err != nil {
			http.Error(w, `{"error": "failed to fetch matches"}`, http.StatusInternalServerError)
			return
		}

		writeJSON(w, repoDetail{
			ID:              repo.ID,
			OwnerName:       repo.OwnerName,
			RepositoryName:  repo.RepositoryName,
			GitProvider:     string(repo.GitProvider),
			LastFetchedAt:   repo.LastFetchedAt,
			DependencyCount: depCount,
			MatchCount:      matchCount,
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
		if len(req.RepositoryName) > 40 || len(req.GitProvider) > 40 || len(req.OwnerName) > 40 {
			http.Error(w, `{"error": "provided information too long"}`, http.StatusBadRequest)
			return
		}
		req.RepositoryName = strings.TrimSpace(req.RepositoryName)
		req.OwnerName = strings.TrimSpace(req.OwnerName)
		req.GitProvider = strings.TrimSpace(req.GitProvider)

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
			ID:              watched.ID,
			OwnerName:       watched.OwnerName,
			RepositoryName:  watched.RepositoryName,
			GitProvider:     string(watched.GitProvider),
			LastFetchedAt:   watched.LastFetchedAt,
			DependencyCount: 0,
			MatchCount:      0,
		})
	}
}

func GetRepositoryMatchesHandler(matchRepo MatchRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		page, limit, err := getPaginationParams(r, 20)
		if err != nil {
			http.Error(w, `{"error": "invalid pagination parameters"}`, http.StatusBadRequest)
			return
		}

		matches, total, err := matchRepo.GetByRepositoryIDPaginated(id, page, limit)
		if err != nil {
			http.Error(w, `{"error": "failed to fetch matches"}`, http.StatusInternalServerError)
			return
		}

		matchesResponse := make([]matchResponse, len(matches))
		for i, m := range matches {
			matchesResponse[i] = toMatchResponse(m)
		}

		totalPages := (total + limit - 1) / limit
		writeJSON(w, paginatedMatchesResponse{
			Items:      matchesResponse,
			Total:      total,
			Page:       page,
			Limit:      limit,
			TotalPages: totalPages,
		})
	}
}

func GetRepositoryDependenciesHandler(depRepo DepRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		page, limit, err := getPaginationParams(r, 50)
		if err != nil {
			http.Error(w, `{"error": "invalid pagination parameters"}`, http.StatusBadRequest)
			return
		}

		deps, total, err := depRepo.GetByRepoIDPaginated(id, page, limit)
		if err != nil {
			http.Error(w, `{"error": "failed to fetch dependencies"}`, http.StatusInternalServerError)
			return
		}

		depsResponse := make([]repositoryDependencyResponse, len(deps))
		for i, d := range deps {
			depsResponse[i] = toRepositoryDependencyResponse(d)
		}

		totalPages := (total + limit - 1) / limit
		writeJSON(w, paginatedDependenciesResponse{
			Items:      depsResponse,
			Total:      total,
			Page:       page,
			Limit:      limit,
			TotalPages: totalPages,
		})
	}
}
