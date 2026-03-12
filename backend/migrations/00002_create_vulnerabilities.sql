-- +goose Up
CREATE TABLE vulnerabilities (
    id TEXT PRIMARY KEY,
    cve TEXT, 
    severity TEXT DEFAULT '',
    description TEXT, 
    source_article_ids TEXT[], 
    published_at TIMESTAMPTZ, 
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()   
);

CREATE UNIQUE INDEX vulnerabilities_cve_unique 
ON vulnerabilities (cve) 
WHERE cve != '';

-- +goose Down
DROP TABLE vulnerabilities;
