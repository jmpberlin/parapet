package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

type MatchDetailRepository interface {
	GetByID(id string) (*domain.Match, error)
}

type MatchVulnerabilityRepository interface {
	GetByID(id string) (*domain.Vulnerability, error)
}

type MatchArticleRepository interface {
	GetByID(id string) (*domain.Article, error)
}

type dependencyResponse struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	PURL    string `json:"purl"`
}

type matchDetailVulnerabilityResponse struct {
	ID                   string                       `json:"id"`
	CVE                  string                       `json:"cve"`
	Severity             string                       `json:"severity"`
	Description          string                       `json:"description"`
	AffectedTechnologies []affectedTechnologyResponse `json:"affected_technologies"`
	PublishedAt          time.Time                    `json:"published_at"`
	CreatedAt            time.Time                    `json:"created_at"`
}

type sourceArticleResponse struct {
	ID          string    `json:"id"`
	Headline    string    `json:"headline"`
	SourceURL   string    `json:"source_url"`
	HostDomain  string    `json:"host_domain"`
	PublishedAt time.Time `json:"published_at"`
}

type matchDetailResponse struct {
	Match          matchResponse                    `json:"match"`
	Dependency     dependencyResponse               `json:"dependency"`
	Vulnerability  matchDetailVulnerabilityResponse `json:"vulnerability"`
	SourceArticles []sourceArticleResponse          `json:"source_articles"`
}

func GetMatchDetailHandler(
	matchRepo MatchDetailRepository,
	vulnRepo MatchVulnerabilityRepository,
	articleRepo MatchArticleRepository,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		match, err := matchRepo.GetByID(id)
		if err != nil {
			http.Error(w, `{"error": "failed to fetch match"}`, http.StatusInternalServerError)
			return
		}
		if match == nil {
			http.Error(w, `{"error": "match not found"}`, http.StatusNotFound)
			return
		}

		vuln := &domain.Vulnerability{}
		if match.VulnerabilityID != "" {
			v, err := vulnRepo.GetByID(match.VulnerabilityID)
			if err != nil {
				http.Error(w, `{"error": "failed to fetch vulnerability"}`, http.StatusInternalServerError)
				return
			}
			if v != nil {
				vuln = v
			}
		}

		articles := make([]sourceArticleResponse, 0, len(vuln.SourceArticleIDs))
		for _, articleID := range vuln.SourceArticleIDs {
			a, err := articleRepo.GetByID(articleID)
			if err != nil {
				http.Error(w, `{"error": "failed to fetch source article"}`, http.StatusInternalServerError)
				return
			}
			if a == nil {
				continue
			}
			articles = append(articles, sourceArticleResponse{
				ID:          a.ID,
				Headline:    a.Headline,
				SourceURL:   a.SourceURL,
				HostDomain:  string(a.HostDomain),
				PublishedAt: a.PublishedAt,
			})
		}

		technologies := make([]affectedTechnologyResponse, len(vuln.AffectedTechnologies))
		for i, t := range vuln.AffectedTechnologies {
			technologies[i] = affectedTechnologyResponse{
				Name:         t.Name,
				PURL:         t.PURL,
				VersionRange: t.VersionRange,
			}
		}

		writeJSON(w, matchDetailResponse{
			Match: toMatchResponse(*match),
			Dependency: dependencyResponse{
				Name:    match.MatchedComponent,
				Version: match.MatchedVersion,
				PURL:    match.ComponentPURL,
			},
			Vulnerability: matchDetailVulnerabilityResponse{
				ID:                   vuln.ID,
				CVE:                  vuln.CVE,
				Severity:             string(vuln.Severity),
				Description:          vuln.Description,
				AffectedTechnologies: technologies,
				PublishedAt:          vuln.PublishedAt,
				CreatedAt:            vuln.CreatedAt,
			},
			SourceArticles: articles,
		})
	}
}
