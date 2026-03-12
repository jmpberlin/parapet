package usecase

import "time"

type PipelineStage string

const (
	StageHarvest    PipelineStage = "harvest"
	StageExtract    PipelineStage = "extract"
	StageUpdateDeps PipelineStage = "update_deps"
	StageMatch      PipelineStage = "match"
)

type PipelineError struct {
	Stage   PipelineStage
	Message string
	Err     error
}

type PipelineResult struct {
	ArticlesHarvested   int
	VulnsExtracted      int
	DependenciesFetched int
	MatchesFound        int
	RanAt               time.Time
	Errors              []PipelineError
}

func (r *PipelineResult) AddError(stage PipelineStage, msg string, err error) {
	r.Errors = append(r.Errors, PipelineError{
		Stage:   stage,
		Message: msg,
		Err:     err,
	})
}

func (r *PipelineResult) HasErrors() bool {
	return len(r.Errors) > 0
}

type Pipeline struct {
	harvest   *HarvestArticlesUseCase
	extract   *ExtractVulnerabilitiesUseCase
	fetchDeps *UpdateDependenciesUseCase
	// match     *MatchVulnerabilitiesUseCase
}
