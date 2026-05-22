-- +goose Up
ALTER TABLE matches ADD COLUMN IF NOT EXISTS confidence TEXT;
ALTER TABLE matches ADD COLUMN IF NOT EXISTS matched_on TEXT;
ALTER TABLE matches ADD COLUMN IF NOT EXISTS vuln_identifier TEXT;
ALTER TABLE matches ADD COLUMN IF NOT EXISTS dep_identifier TEXT;

-- +goose Down
ALTER TABLE matches DROP COLUMN IF EXISTS confidence;
ALTER TABLE matches DROP COLUMN IF EXISTS matched_on;
ALTER TABLE matches DROP COLUMN IF EXISTS vuln_identifier;
ALTER TABLE matches DROP COLUMN IF EXISTS dep_identifier;
