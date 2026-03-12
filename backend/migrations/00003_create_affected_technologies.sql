-- +goose Up
CREATE TABLE affected_technologies (
    id TEXT PRIMARY KEY,
    vulnerability_id TEXT NOT NULL REFERENCES vulnerabilities(id),
    name TEXT NOT NULL,
    purl TEXT,
    version_range TEXT
);

CREATE UNIQUE INDEX affected_technologies_vuln_purl_unique
ON affected_technologies (vulnerability_id, purl)
WHERE purl != '';

-- +goose Down
DROP TABLE affected_technologies;