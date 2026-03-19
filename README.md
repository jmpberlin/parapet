# Parapet

Parapet monitors security news and alerts you when a vulnerability affects a library your project depends on.

Parapet crawls streams of news like bleeping computer or hacker news, uses an LLM to extract vulnerabilities and the software components they affect, then cross-references those components against the dependency manifest of your GitHub repository.

## How it works

The pipeline runs four stages on a schedule:

1. **Harvest** — crawls security news sources (BleepingComputer, Hacker News) for new articles
2. **Extract** — sends article content to the Claude API which extracts vulnerabilities, CVEs, affected technologies and severity using structured tool use
3. **Update dependencies** — fetches your repository's dependencies from GitHub's SBOM endpoint
4. **Match** — cross-references extracted vulnerabilities against your dependencies using PURL and name matching

## Tech stack

**Backend** — Go, PostgreSQL, Chi router, Goose migrations, Testcontainers

**Frontend** — React, TypeScript, React Query, SCSS, OpenAPI-generated types

**Infrastructure** — Hetzner VPS, Docker Compose, Caddy

**Integrations** — Anthropic Claude API, GitHub Dependency Graph API

## Architecture

The backend follows a clean layered architecture — domain, usecase, adapter, repository. Each layer communicates through interfaces. Business logic lives in the usecase layer and has no knowledge of the database, HTTP, or external APIs.

## Status

Work in progress. This is a portfolio project — the pipeline runs and data flows end to end. Known limitations: version range matching is not yet implemented, and name-based dependency matching is fuzzy by nature. NVD integration and improved matching are on the roadmap.

## Running Locally

Copy `.env.example` to `.env` and fill in the required values: Postgres credentials, a Claude API key, and a GitHub fine-grained personal access token. Then start the stack with Docker Compose:

\```
docker compose up --build
\```

The backend will be available at `http://localhost:8080` and the Swagger UI at `http://localhost:8080/docs/`. The frontend dev server is run separately:

\```
cd frontend && npm install && npm run dev
\```

It will be available at `http://localhost:5173`.

## Technical Notes

### GitHub Integration

**Fine-grained token permissions**
The GitHub fine-grained personal access token must have **Contents: Read** permission to access repository data.

**Dependency graph endpoint**
The dependency graph API endpoint must be explicitly enabled on each individual repository under:
`Repository Settings > Advanced security > Dependency graph > enable`

If disabled, requests to the dependency graph endpoint will return `404`.