package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/jmpberlin/nightwatch/backend/internal/usecase"
)

const maxLookbackDays = 30

type PipelineRunner interface {
	Run(customLookback time.Duration)
	LastResult() *usecase.PipelineResult
	IsRunning() bool
}

type pipelineErrorResponse struct {
	Stage string `json:"stage"`
	Error string `json:"error"`
}

type pipelineResultResponse struct {
	RanAt                    time.Time               `json:"ran_at"`
	RunInProgress            bool                    `json:"run_in_progress"`
	ArticlesHarvested        int                     `json:"articles_harvested"`
	VulnerabilitiesExtracted int                     `json:"vulnerabilities_extracted"`
	DepsAdded                int                     `json:"deps_added"`
	DepsRemoved              int                     `json:"deps_removed"`
	MatchesFound             int                     `json:"matches_found"`
	Errors                   []pipelineErrorResponse `json:"errors"`
}

func toPipelineResultResponse(r usecase.PipelineResult) pipelineResultResponse {
	errors := make([]pipelineErrorResponse, len(r.Errors))
	for i, e := range r.Errors {
		errors[i] = pipelineErrorResponse{
			Stage: string(e.Stage),
			Error: e.Err.Error(),
		}
	}
	return pipelineResultResponse{
		RanAt:                    r.RanAt,
		RunInProgress:            r.RunInProgress,
		ArticlesHarvested:        r.ArticlesHarvested,
		VulnerabilitiesExtracted: r.VulnerabilitiesExtracted,
		DepsAdded:                r.DepsAdded,
		DepsRemoved:              r.DepsRemoved,
		MatchesFound:             r.MatchesFound,
		Errors:                   errors,
	}
}

func PipelineRunHandler(pipeline PipelineRunner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if pipeline.IsRunning() {
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(`{"error": "pipeline already running"}`))
			return
		}

		var customLookback time.Duration
		if raw := r.URL.Query().Get("days"); raw != "" {
			days, err := strconv.Atoi(raw)
			if err != nil || days < 1 {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "days must be a positive integer"}`))
				return
			}
			if days > maxLookbackDays {
				days = maxLookbackDays
			}
			customLookback = time.Duration(days) * 24 * time.Hour
		}

		go pipeline.Run(customLookback)
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status": "pipeline started"}`))
	}
}

func PipelineStatusHandler(pipeline PipelineRunner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result := pipeline.LastResult()
		if result == nil {
			writeJSON(w, toPipelineResultResponse(usecase.PipelineResult{}))
			return
		}
		writeJSON(w, toPipelineResultResponse(*result))
	}
}
