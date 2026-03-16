package handler

import (
	"net/http"

	"github.com/jmpberlin/nightwatch/backend/internal/usecase"
)

type PipelineRunner interface {
	Run()
	LastResult() *usecase.PipelineResult
}

func PipelineRunHandler(pipeline PipelineRunner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		go pipeline.Run()
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status": "pipeline started"}`))
	}
}

func PipelineStatusHandler(pipeline PipelineRunner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result := pipeline.LastResult()
		if result == nil {
			writeJSON(w, map[string]string{"status": "no pipeline run yet"})
			return
		}
		writeJSON(w, result)
	}
}
