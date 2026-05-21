package usecase

import (
	"log/slog"
	"time"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

type HarvestResult struct {
	ArticlesHarvested int
	Errors            []error
}

type HarvestArticlesUseCase struct {
	articleRepo         ArticleRepository
	crawlerOrchestrator CrawlerOrchestrator
	defaultLookback     time.Duration
}

func NewHarvestArticlesUseCase(articleRepo ArticleRepository, crawlerOrchestrator CrawlerOrchestrator, defaultLookback time.Duration) *HarvestArticlesUseCase {
	return &HarvestArticlesUseCase{
		articleRepo:         articleRepo,
		crawlerOrchestrator: crawlerOrchestrator,
		defaultLookback:     defaultLookback,
	}
}

func (h *HarvestArticlesUseCase) Execute(customLookback time.Duration) HarvestResult {
	lookback := h.defaultLookback
	if customLookback > 0 {
		lookback = customLookback
	}
	since := time.Now().Add(-lookback)
	result := HarvestResult{}
	articles, errs := h.crawlerOrchestrator.FetchArticles(since)
	for _, err := range errs {
		slog.Error("scraper failed", "stage", StageHarvest, "err", err)
		result.Errors = append(result.Errors, err)
	}
	if len(articles) == 0 {
		return result
	}

	for _, article := range articles {
		existing, err := h.articleRepo.GetByURL(article.SourceURL)
		if err != nil {
			slog.Error("failed to check article existence", "stage", StageHarvest, "url", article.SourceURL, "err", err)
			result.Errors = append(result.Errors, err)
			continue
		}
		if existing != nil {
			continue
		}

		article.ID = domain.NewID()
		if err := h.articleRepo.Save(article); err != nil {
			slog.Error("failed to save article", "stage", StageHarvest, "url", article.SourceURL, "err", err)
			result.Errors = append(result.Errors, err)
			continue
		}
		result.ArticlesHarvested++
	}
	slog.Info("harvest complete",
		"stage", StageHarvest,
		"articles_harvested", result.ArticlesHarvested,
		"errors", len(result.Errors),
	)

	return result
}
