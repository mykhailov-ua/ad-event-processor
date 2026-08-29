-- +goose Up
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS traffic_template_id TEXT,
    ADD COLUMN IF NOT EXISTS click_query_params JSONB NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE campaigns
    DROP COLUMN IF EXISTS click_query_params,
    DROP COLUMN IF EXISTS traffic_template_id;
