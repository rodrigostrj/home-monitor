-- +goose Up
ALTER TABLE sources DROP COLUMN api_key;

-- +goose Down
ALTER TABLE sources ADD COLUMN api_key TEXT NOT NULL DEFAULT '';
