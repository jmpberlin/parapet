-- +goose Up
CREATE TABLE affected_technologies (
    id TEXT PRIMARY KEY,
    vulnerability_id TEXT NOT NULL REFERENCES vulnerabilities(id),
    name TEXT NOT NULL,
    purl TEXT,
    version_range TEXT
);

-- +goose Down
DROP TABLE affected_technologies;