-- +goose Up
CREATE TABLE vulnerabilities (
    id TEXT PRIMARY KEY,
    cve TEXT UNIQUE, 
    severity TEXT DEFAULT '',
    description TEXT, 
    source_article_ids TEXT[], 
    published_at TIMESTAMPTZ, 
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()   
);

-- +goose Down
DROP TABLE vulnerabilities;
