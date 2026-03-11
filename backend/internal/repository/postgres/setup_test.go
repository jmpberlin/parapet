package postgres_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/jmpberlin/nightwatch/backend/migrations"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx,
		"postgres:16",
		tcpostgres.WithDatabase("nightwatch_test"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp"),
		),
	)
	if err != nil {
		log.Fatalf("failed to start postgres container: %s", err)
	}
	defer container.Terminate(ctx)

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("failed to get connection string: %s", err)
	}

	testDB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("failed to open test db: %s", err)
	}

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("failed to set goose dialect: %s", err)
	}
	if err := goose.Up(testDB, "."); err != nil {
		log.Fatalf("failed to run migrations: %s", err)
	}

	os.Exit(m.Run())
}

func truncateTables(t *testing.T) {
	t.Helper()
	_, err := testDB.Exec(`TRUNCATE TABLE affected_technologies, vulnerabilities CASCADE`)
	if err != nil {
		t.Fatalf("failed to truncate tables: %s", err)
	}
}
