-- +goose Up
CREATE TYPE match_status AS ENUM ('CONFIRMED', 'WARNING', 'RESOLVED');

CREATE TABLE matches (
    id TEXT PRIMARY KEY,
    vulnerability_id TEXT NOT NULL REFERENCES vulnerabilities(id),
    repository_id TEXT NOT NULL REFERENCES watched_repositories(id),
    component_purl TEXT,
    matched_component TEXT NOT NULL,
    matched_version TEXT,
    status match_status NOT NULL DEFAULT 'WARNING',
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX matches_unique_with_purl
ON matches (vulnerability_id, repository_id, component_purl)
WHERE component_purl != '';

CREATE UNIQUE INDEX matches_unique_without_purl
ON matches (vulnerability_id, repository_id, matched_component)
WHERE component_purl = '';

-- +goose Down
DROP TABLE matches;
DROP TYPE match_status;