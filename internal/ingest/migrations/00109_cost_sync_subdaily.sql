-- +goose Up
ALTER TABLE cost_sync_credentials
    ADD COLUMN IF NOT EXISTS sync_interval_minutes INT NOT NULL DEFAULT 1440,
    ADD COLUMN IF NOT EXISTS token_mapping JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS next_run_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS cost_sync_attribution_applied (
    sync_run_id BIGINT NOT NULL REFERENCES cost_sync_runs(id) ON DELETE CASCADE,
    campaign_id UUID NOT NULL,
    placement_id TEXT NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (sync_run_id, campaign_id, placement_id)
);

-- +goose Down
DROP TABLE IF EXISTS cost_sync_attribution_applied;
ALTER TABLE cost_sync_credentials
    DROP COLUMN IF EXISTS next_run_at,
    DROP COLUMN IF EXISTS token_mapping,
    DROP COLUMN IF EXISTS sync_interval_minutes;
