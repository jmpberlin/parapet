-- +goose Up
CREATE TABLE articles (
    id TEXT PRIMARY KEY,
    source_url TEXT NOT NULL UNIQUE,
    host_domain TEXT NOT NULL,
    headline TEXT NOT NULL,
    author TEXT,
    content_html TEXT,
    content_cleaned TEXT,
    published_at TIMESTAMPTZ NOT NULL,
    crawled_at TIMESTAMPTZ NOT NULL,
    processed_at TIMESTAMPTZ
);

-- +goose Down
DROP TABLE articles;