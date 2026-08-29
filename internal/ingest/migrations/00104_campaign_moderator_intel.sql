-- +goose Up
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS moderator_intel_enabled BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE campaigns
    DROP COLUMN IF EXISTS moderator_intel_enabled;
