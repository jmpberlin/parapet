-- +goose Up
ALTER TABLE matches ADD COLUMN IF NOT EXISTS version_check TEXT;

-- +goose Down
ALTER TABLE matches DROP COLUMN IF EXISTS version_check;
