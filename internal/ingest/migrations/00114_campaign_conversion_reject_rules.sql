-- +goose Up
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS conversion_reject_rules JSONB NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE campaigns DROP COLUMN IF EXISTS conversion_reject_rules;
