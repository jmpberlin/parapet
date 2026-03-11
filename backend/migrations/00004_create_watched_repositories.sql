-- +goose Up
CREATE TABLE watched_repositories (
    id TEXT PRIMARY KEY,
    git_provider TEXT NOT NULL, 
    owner_name TEXT NOT NULL, 
    repository_name TEXT NOT NULL, 
    integrated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), 
    archived_at TIMESTAMPTZ,
    last_scanned_at TIMESTAMPTZ
);

-- +goose Down
DROP TABLE watched_repositories;