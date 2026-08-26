-- +goose Up
CREATE TABLE IF NOT EXISTS campaign_conversion_mappings (
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    inbound_status TEXT NOT NULL,
    goal_name TEXT NOT NULL DEFAULT '',
    payout_micro BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (campaign_id, inbound_status)
);

CREATE INDEX IF NOT EXISTS idx_campaign_conversion_mappings_campaign
    ON campaign_conversion_mappings(campaign_id);

-- +goose Down
DROP TABLE IF EXISTS campaign_conversion_mappings;
