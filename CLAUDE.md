# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this project is

Parapet monitors security news and alerts when a vulnerability affects a library a tracked GitHub repository depends on. It runs a four-stage pipeline on a schedule: **Harvest → Extract → UpdateDeps → Match**.

- **Harvest** — crawls BleepingComputer and Socket.dev for new articles
- **Extract** — sends article content to an LLM (Anthropic or any OpenAI-compatible API such as TensorX, selected via `LLM_PROVIDER`) with a forced `extract_vulnerabilities` tool call to extract vulnerabilities, CVEs, affected technologies, and severity
- **UpdateDeps** — fetches repository dependencies from GitHub's SBOM/dependency graph endpoint
- **Match** — cross-references extracted vulnerabilities against watched-repository dependencies using a four-tier matching algorithm (PURL → namespace → package name → keyword), then applies version range checks via semver

## Development commands

**Backend (Go):**
```bash
cd backend
go build ./...
go test ./...                          # runs all tests; repo tests spin up a real Postgres via Testcontainers
go test ./internal/repository/...     # run only repository tests
go test ./internal/usecase/...        # run only usecase tests
go test -run TestFunctionName ./...   # run a single test
```

**Frontend (React/TypeScript):**
```bash
cd frontend
npm install
npm run dev        # dev server at http://localhost:5173
npm run build      # tsc + vite build
npm run lint       # eslint
npm test           # vitest (watch mode)
npm run test:run   # vitest (single run)
npm run generate:types  # regenerate src/types/api.ts from ../backend/openapi.yaml
```

**Full stack (Docker Compose):**
```bash
cp .env.example .env   # fill in POSTGRES_*, LLM_*, GITHUB_TOKEN
docker compose up --build
# backend at http://localhost:8080, Swagger at http://localhost:8080/docs/
```

## Architecture

### Backend (`backend/`)

Clean layered architecture — each layer communicates through interfaces defined in `internal/usecase/interfaces.go`. Business logic has no knowledge of HTTP, database, or external APIs.

```
cmd/api/main.go              — wires everything together: DB, adapters, use cases, router, scheduler
internal/
  domain/                    — pure data types: Article, Vulnerability, AffectedTechnology, Match, WatchedRepository, RepositoryDependency
  usecase/                   — business logic; depends only on interfaces and domain
    pipeline.go              — orchestrates the four stages; atomic run-guard prevents concurrent runs
    harvest_articles.go      — deduplicates articles by URL before saving
    extract_vulnerabilities.go
    update_dependencies.go   — diffs old vs new deps, saves additions, deletes removals
    match_vulnerabilities.go — iterates all repos × unmatched vulns × affected technologies
    match_tiers.go           — the four-tier matching logic + version range check (Masterminds/semver)
    normalize.go             — PURL parsing and keyword tokenisation; NormalizedIdentifier is central to matching
    interfaces.go            — all repository + adapter interfaces used by use cases
  adapter/
    llm/                     — vulnerability extraction via LLMs
      llm.go                 — ConfigFromEnv (LLM_* vars) + New(cfg): picks the provider adapter
      shared/                — provider-independent: system prompt, tool JSON schema, payload DTO → domain conversion, 5-way concurrent fan-out
      anthropic/             — Anthropic extractor (anthropic-sdk-go, forced tool use)
      openai/                — OpenAI-standard extractor for any compatible API, e.g. TensorX (openai-go SDK + base URL, forced function call)
    crawler/                 — BCScraper and SocketDevScraper implement SourceScraper; CrawlerOrchestrator fans out
    github/client.go         — fetches SBOM from GitHub dependency graph API
  repository/postgres/       — SQL implementations of all usecase interfaces; migrations embedded via go:embed
  handler/                   — thin Chi HTTP handlers; each handler function closes over the use case or repo it needs
migrations/                  — Goose SQL migrations (embedded into the binary via migrations.go)
```

**Key cross-cutting facts:**
- The pipeline runs once on startup, then every hour via `time.Ticker`
- Matching skips articles older than 48 hours during harvest (configurable lookback)
- Repository tests use Testcontainers (real Postgres spun up per `TestMain`); no mocks for DB
- The Go module is named `github.com/jmpberlin/nightwatch/backend` (historical name, not "parapet")

### Frontend (`frontend/`)

Vite + React 19 + TypeScript. State: React Query for server state, Zustand for client state.

```
src/
  api/           — one file per resource (articles, vulnerabilities, repositories, pipeline); all use apiFetch from client.ts
  pages/         — one folder per route (Landing, Articles, ArticleDetail, Vulnerabilities, VulnerabilityDetail, Repositories)
  components/    — shared components
  hooks/         — shared hooks
  types/         — api.ts is auto-generated from backend OpenAPI spec (don't edit manually)
```

API base URL is `VITE_API_URL` env var, defaulting to `/api` (Caddy strips the prefix before proxying to the backend).

### Infrastructure

Docker Compose runs: `postgres`, `backend`, `loki`, `promtail`, `grafana`, `adminer`. Two Docker networks: `app-network` (postgres + backend + adminer) and `observability-network` (loki + promtail + grafana). Adminer binds only to `127.0.0.1:8081`. Grafana at `127.0.0.1:3000`.

Production: Hetzner VPS, Caddy as reverse proxy (auto TLS). CI/CD via GitHub Actions on push to `main` — builds frontend, scps dist to server, pulls and rebuilds backend container.

## Environment variables

See `.env.example`. Required: `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `GITHUB_TOKEN`, and the extraction LLM settings `LLM_PROVIDER` (`anthropic` | `openai`), `LLM_API_KEY`, `LLM_MODEL`, plus `LLM_BASE_URL` for `openai` — documented and validated in `backend/internal/adapter/llm/llm.go`. Optional: `BACKEND_PORT`, `BACKEND_HOST`, `BACKEND_HOST_PORT`, `GRAFANA_PASSWORD`.
