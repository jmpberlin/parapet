-- +goose Up
CREATE TABLE repository_dependencies (
    id TEXT PRIMARY KEY, 
    repository_id TEXT NOT NULL REFERENCES watched_repositories(id),
    name TEXT NOT NULL, 
    version TEXT, 
    purl TEXT, 
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE repository_dependencies;