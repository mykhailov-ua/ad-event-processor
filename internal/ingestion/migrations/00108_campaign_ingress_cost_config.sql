-- +goose Up
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS ingress_cost_config JSONB NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE campaigns
    DROP COLUMN IF EXISTS ingress_cost_config;
