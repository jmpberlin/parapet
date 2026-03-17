package usecase

import (
	"log/slog"
	"sync"
	"time"
)

type PipelineStage string

const (
	StageHarvest    PipelineStage = "harvest"
	StageExtract    PipelineStage = "extract"
	StageUpdateDeps PipelineStage = "update_deps"
	StageMatch      PipelineStage = "match"
)

type PipelineError struct {
	Stage PipelineStage
	Err   error
}

func (e PipelineError) Error() string {
	return e.Err.Error()
}

type PipelineResult struct {
	RanAt                    time.Time
	ArticlesHarvested        int
	VulnerabilitiesExtracted int
	DepsAdded                int
	DepsRemoved              int
	MatchesFound             int
	RunInProgress            bool
	Errors                   []PipelineError
}

func (r *PipelineResult) HasErrors() bool {
	return len(r.Errors) > 0
}

type Pipeline struct {
	harvest    *HarvestArticlesUseCase
	extract    *ExtractVulnerabilitiesUseCase
	updateDeps *UpdateDependenciesUseCase
	match      *MatchVulnerabilitiesUseCase
	mu         sync.RWMutex
	lastResult *PipelineResult
}

func NewPipeline(harvest *HarvestArticlesUseCase, extract *ExtractVulnerabilitiesUseCase, updateDeps *UpdateDependenciesUseCase, match *MatchVulnerabilitiesUseCase) *Pipeline {
	return &Pipeline{
		harvest:    harvest,
		extract:    extract,
		updateDeps: updateDeps,
		match:      match,
	}
}

func (p *Pipeline) LastResult() *PipelineResult {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lastResult
}

func (p *Pipeline) Run() {
	result := PipelineResult{RanAt: time.Now(), RunInProgress: true}
	p.updateLastResult(&result)

	harvestResult := p.harvest.Execute()
	result.ArticlesHarvested = harvestResult.ArticlesHarvested
	for _, err := range harvestResult.Errors {
		result.Errors = append(result.Errors, PipelineError{Stage: StageHarvest, Err: err})
	}
	p.updateLastResult(&result)

	extractResult := p.extract.Execute()
	result.VulnerabilitiesExtracted = extractResult.VulnerabilitiesExtracted
	for _, err := range extractResult.Errors {
		result.Errors = append(result.Errors, PipelineError{Stage: StageExtract, Err: err})
	}
	p.updateLastResult(&result)

	updateDepsResult := p.updateDeps.Execute()
	result.DepsAdded = updateDepsResult.DepsAdded
	result.DepsRemoved = updateDepsResult.DepsRemoved
	for _, err := range updateDepsResult.Errors {
		result.Errors = append(result.Errors, PipelineError{Stage: StageUpdateDeps, Err: err})
	}
	p.updateLastResult(&result)

	matchResult := p.match.Execute()
	result.MatchesFound = matchResult.MatchesFound
	for _, err := range matchResult.Errors {
		result.Errors = append(result.Errors, PipelineError{Stage: StageMatch, Err: err})
	}
	result.RunInProgress = false
	p.updateLastResult(&result)

	slog.Info("pipeline run complete",
		"articles_harvested", result.ArticlesHarvested,
		"vulnerabilities_extracted", result.VulnerabilitiesExtracted,
		"deps_added", result.DepsAdded,
		"deps_removed", result.DepsRemoved,
		"matches_found", result.MatchesFound,
		"total_errors", len(result.Errors),
	)
}

func (p *Pipeline) updateLastResult(result *PipelineResult) {
	p.mu.Lock()
	p.lastResult = result
	p.mu.Unlock()
}
