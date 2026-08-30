-- +goose Up
ALTER TABLE flows
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- +goose Down
ALTER TABLE flows DROP COLUMN IF EXISTS updated_at;
