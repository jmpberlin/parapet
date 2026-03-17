-- +goose Up
CREATE TABLE watched_repositories (
    id TEXT PRIMARY KEY,
    git_provider TEXT NOT NULL, 
    owner_name TEXT NOT NULL, 
    repository_name TEXT NOT NULL, 
    integrated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), 
    archived_at TIMESTAMPTZ,
    last_fetched_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX watched_repositories_owner_name_provider_unique
ON watched_repositories (owner_name, git_provider, repository_name)
WHERE repository_name != '' AND owner_name != '' AND git_provider != '';

-- +goose Down
DROP TABLE watched_repositories;