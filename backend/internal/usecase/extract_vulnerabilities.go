package usecase

import (
	"fmt"
	"log/slog"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

type ExtractionStageResult struct {
	VulnerabilitiesExtracted int
	Errors                   []error
}

type VulnerabilityExtractionUseCase struct {
	vulnerabilityRepo VulnerabilityRepository
	articleRepo       ArticleRepository
	extractor         VulnerabilityExtractor
}

func NewVulnerabilityExtractionUseCase(vulnerabilityRepo VulnerabilityRepository, articleRepo ArticleRepository, extractor VulnerabilityExtractor) *VulnerabilityExtractionUseCase {
	return &VulnerabilityExtractionUseCase{
		vulnerabilityRepo: vulnerabilityRepo,
		articleRepo:       articleRepo,
		extractor:         extractor,
	}
}

func (v *VulnerabilityExtractionUseCase) Execute() ExtractionStageResult {
	result := ExtractionStageResult{}

	articles, err := v.articleRepo.GetUnprocessed()
	if err != nil {
		slog.Error("failed to fetch unprocessed articles",
			"stage", StageExtract,
			"err", err,
		)
		result.Errors = append(result.Errors, fmt.Errorf("%s: fetch unprocessed articles failed: %w", StageExtract, err))
		return result
	}

	if len(articles) == 0 {
		slog.Info("no unprocessed articles found", "stage", StageExtract)
		return result
	}

	extractionResults := v.extractor.ExtractVulnerabilities(articles)
	for _, er := range extractionResults {
		saved, errs := v.processExtractionResult(er)
		result.VulnerabilitiesExtracted += saved
		result.Errors = append(result.Errors, errs...)
	}

	slog.Info("extraction stage complete",
		"stage", StageExtract,
		"vulnerabilities_extracted", result.VulnerabilitiesExtracted,
		"errors", len(result.Errors),
	)
	return result
}

func (v *VulnerabilityExtractionUseCase) processExtractionResult(er domain.ArticleExtractionResult) (int, []error) {
	var errs []error

	if er.Err != nil {
		slog.Error("failed to extract vulnerabilities from article",
			"stage", StageExtract,
			"article_id", er.ArticleID,
			"err", er.Err,
		)
		errs = append(errs, fmt.Errorf("%s: extract article %s: %w", StageExtract, er.ArticleID, er.Err))
		return 0, errs
	}

	if len(er.Vulnerabilities) == 0 {
		slog.Info("no vulnerabilities found in article",
			"stage", StageExtract,
			"article_id", er.ArticleID,
		)
		if err := v.setArticleToProcessed(er.ArticleID); err != nil {
			errs = append(errs, err)
		}
		return 0, errs
	}

	saved, saveErrs := v.saveVulnerabilities(er.Vulnerabilities)
	errs = append(errs, saveErrs...)

	if len(saveErrs) > 0 {
		slog.Warn("skipping mark processed — not all vulnerabilities saved",
			"stage", StageExtract,
			"article_id", er.ArticleID,
			"saved", saved,
			"total", len(er.Vulnerabilities),
		)
		return saved, errs
	}

	if err := v.setArticleToProcessed(er.ArticleID); err != nil {
		errs = append(errs, err)
	}
	return saved, errs
}

func (v *VulnerabilityExtractionUseCase) saveVulnerabilities(vulns []domain.Vulnerability) (int, []error) {
	var errs []error
	saved := 0

	for _, vuln := range vulns {
		vuln.ID = domain.NewID()
		if err := v.vulnerabilityRepo.Save(vuln); err != nil {
			slog.Error("failed to save vulnerability",
				"stage", StageExtract,
				"cve", vuln.CVE,
				"err", err,
			)
			errs = append(errs, fmt.Errorf("%s: save vulnerability %s failed: %w", StageExtract, vuln.CVE, err))
			continue
		}
		saved++
	}
	return saved, errs
}

func (v *VulnerabilityExtractionUseCase) setArticleToProcessed(id string) error {
	if err := v.articleRepo.MarkProcessed(id); err != nil {
		slog.Error("failed to mark article as processed",
			"stage", StageExtract,
			"article_id", id,
			"err", err,
		)
		return fmt.Errorf("%s: mark article %s as processed: %w", StageExtract, id, err)
	}
	return nil
}
