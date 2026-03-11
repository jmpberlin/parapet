-- +goose Up
CREATE TABLE affected_technologies (
    id TEXT PRIMARY KEY,
    vulnerability_id TEXT NOT NULL REFERENCES vulnerabilities(id),
    name TEXT NOT NULL,
    purl TEXT,
    version_range TEXT,
    CONSTRAINT affected_technologies_vulnerability_purl_unique 
        UNIQUE (vulnerability_id, purl)
);

-- +goose Down
DROP TABLE affected_technologies;