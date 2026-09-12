-- +goose Up
CREATE TABLE IF NOT EXISTS campaign_status_scheme_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    sort_order INT NOT NULL,
    when_status TEXT NOT NULL DEFAULT '',
    when_goal TEXT NOT NULL DEFAULT '',
    set_internal_status TEXT NOT NULL DEFAULT '',
    set_goal_name TEXT NOT NULL DEFAULT '',
    payout_mode TEXT NOT NULL DEFAULT 'inherit',
    payout_micro BIGINT NOT NULL DEFAULT 0,
    fire_outbound BOOLEAN NOT NULL DEFAULT TRUE,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (campaign_id, sort_order)
);

CREATE INDEX IF NOT EXISTS idx_campaign_status_scheme_rules_campaign
    ON campaign_status_scheme_rules(campaign_id);

-- +goose Down
DROP TABLE IF EXISTS campaign_status_scheme_rules;
