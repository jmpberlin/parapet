package main

import (
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jmpberlin/nightwatch/backend/internal/adapter/claude"
	"github.com/jmpberlin/nightwatch/backend/internal/adapter/crawler"
	"github.com/jmpberlin/nightwatch/backend/internal/adapter/github"
	"github.com/jmpberlin/nightwatch/backend/internal/repository/postgres"
	"github.com/jmpberlin/nightwatch/backend/internal/usecase"
	"github.com/jmpberlin/nightwatch/backend/migrations"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

var db *sql.DB

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getPort() string {
	return getEnv("PORT", "8081")
}

func initDB() error {
	host := getEnv("POSTGRES_HOST", "postgres")
	port := getEnv("POSTGRES_PORT", "5432")
	user := getEnv("POSTGRES_USER", "postgres")
	password := getEnv("POSTGRES_PASSWORD", "")
	dbname := getEnv("POSTGRES_DB", "nightwatch")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	slog.Info("connecting to database", "host", host, "port", port, "dbname", dbname)

	var err error
	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", connStr)
		if err == nil {
			err = db.Ping()
			if err == nil {
				slog.Info("database connection established", "attempt", i+1)
				return nil
			}
		}
		slog.Warn("database connection attempt failed",
			"attempt", i+1,
			"max_attempts", 10,
			"err", err,
		)
		time.Sleep(time.Duration(2*(i+1)) * time.Second)
	}
	return fmt.Errorf("failed to connect to database after 10 attempts: %w", err)
}

func runMigrations(db *sql.DB) error {
	goose.SetDialect("postgres")
	goose.SetBaseFS(migrations.FS)
	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("goose migration failed: %w", err)
	}
	slog.Info("database migrations completed successfully")
	return nil
}

func startScheduler(pipeline *usecase.Pipeline) {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		slog.Info("running initial pipeline run")
		pipeline.Run()
		for range ticker.C {
			slog.Info("scheduled pipeline run starting")
			pipeline.Run()
		}
	}()
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := initDB(); err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer db.Close()

	if err := runMigrations(db); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// repositories
	articleRepo := postgres.NewArticleRepository(db)
	vulnRepo := postgres.NewVulnerabilityRepository(db)
	depRepo := postgres.NewDependencyRepository(db)
	watchedRepositoriesRepo := postgres.NewWatchedRepoRepository(db)
	matchRepo := postgres.NewMatchRepository(db)

	// adapters
	bcScraper := crawler.NewBCScraper()
	crawlerOrchestrator := crawler.NewCrawlerOrchestrator([]crawler.SourceScraper{bcScraper})
	claudeClient := claude.NewClaudeClient(getEnv("CLAUDE_API_KEY", ""))
	githubClient := github.NewGithubClient(getEnv("GITHUB_TOKEN", ""))

	// usecases
	harvestUC := usecase.NewHarvestArticlesUseCase(articleRepo, crawlerOrchestrator, 48*time.Hour)
	extractUC := usecase.NewExtractVulnerabilitiesUseCase(vulnRepo, articleRepo, claudeClient)
	updateDepsUC := usecase.NewUpdateDependenciesUseCase(watchedRepositoriesRepo, depRepo, githubClient)
	matchUC := usecase.NewMatchVulnerabilitiesUseCase(watchedRepositoriesRepo, depRepo, vulnRepo, matchRepo)

	// pipeline
	pipeline := usecase.NewPipeline(harvestUC, extractUC, updateDepsUC, matchUC)

	// scheduler
	startScheduler(pipeline)

	// http server
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthcheckHandler)
	mux.HandleFunc("GET /pipeline/status", pipelineStatusHandler(pipeline))

	port := getPort()
	slog.Info("starting http server", "port", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server failed to start: %v", err)
	}
}

func healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status": "ok"}`)
}

func pipelineStatusHandler(pipeline *usecase.Pipeline) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		result := pipeline.LastResult
		if result == nil {
			fmt.Fprintf(w, `{"status": "no pipeline run yet"}`)
			return
		}
		fmt.Fprintf(w, `{"ran_at": "%s", "articles_harvested": %d, "vulnerabilities_extracted": %d, "deps_added": %d, "deps_removed": %d, "matches_found": %d, "total_errors": %d}`,
			result.RanAt.Format(time.RFC3339),
			result.ArticlesHarvested,
			result.VulnerabilitiesExtracted,
			result.DepsAdded,
			result.DepsRemoved,
			result.MatchesFound,
			len(result.Errors),
		)
	}
}
