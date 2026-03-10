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

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

	slog.Info("Connecting to database",
		"host", host,
		"port", port,
		"dbname", dbname,
	)

	var err error
	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", connStr)
		if err == nil {
			err = db.Ping()
			if err == nil {
				slog.Info("Database connection established succesfully",
					"attempt", i+1,
				)
				return nil
			}
		}
		slog.Warn("database connection attempt failed",
			"attempt", i+1,
			"max_attempts", 10,
			"error", err,
		)
		time.Sleep(time.Duration(2*(i+1)) * time.Second)
	}

	return fmt.Errorf("failed to connect to database after 10 attempts: %v", err)
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

func healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	slog.Info("Healthcheck",
		"method", r.Method,
		"path", r.URL.Path,
		"proto", r.Proto,
		"remote_address", r.RemoteAddr,
	)
	fmt.Fprintf(w, `{"status": "ok"}`)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := initDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	if err := runMigrations(db); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	port := getPort()
	http.HandleFunc("/status", healthcheckHandler)
	BCScraper := crawler.NewBCScraper()
	orchestrator := crawler.NewCrawlerOrchestrator([]crawler.SourceScraper{BCScraper}, time.Hour*120)
	claudeApiKey := getEnv("CLAUDE_API_KEY", "")
	vulnerabilityExtractor := claude.NewClaudeClient(claudeApiKey)

	articles, err := orchestrator.FetchAll()
	if err != nil && len(articles) == 0 {
		slog.Error("crawler orchestrator failed to fetch any articles",
			"error", err,
		)
	}
	if err != nil && len(articles) > 0 {
		slog.Warn("at least one crawler collected errors when fetching articles",
			"error", err,
			"collected_articles", articles)
	}
	if err == nil && len(articles) == 0 {
		slog.Warn("crawler orchestrator didn't encounter any errors, but didn't collect any articles - check if target html structure changed")
	}
	vulnerabilities, err := vulnerabilityExtractor.ExtractVulnerabilities(articles)
	if err != nil {
		slog.Warn("when extracting vulnerabilities from articles, the following errors occured",
			"errors", err,
			"articles", articles)
	}
	slog.Info("extracted vulnerabilities",
		"vulnerabilities", vulnerabilities)

	githubOwner := getEnv("GITHUB_OWNER", "")
	githubRepo := getEnv("GITHUB_REPO", "")
	githubToken := getEnv("GITHUB_TOKEN", "")
	githubClient := github.NewGithubClient()
	dependencies, err := githubClient.GetDependencies(githubOwner, githubRepo, githubToken)
	if err != nil {
		slog.Error("github adapter client", "failed to fetch and process list of software components of repo", err)
	}
	slog.Info("successfully fetched and processed dependencies", "dependencies", dependencies)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
