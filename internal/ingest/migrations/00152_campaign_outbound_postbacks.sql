-- +goose Up
CREATE TABLE IF NOT EXISTS campaign_outbound_postbacks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name TEXT NOT NULL DEFAULT '',
    priority INT NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    provider TEXT NOT NULL DEFAULT 'webhook',
    url_template TEXT NOT NULL,
    api_token_encrypted BYTEA NOT NULL DEFAULT '',
    target_event TEXT NOT NULL DEFAULT 'conversion',
    trigger_kind TEXT NOT NULL DEFAULT 'conversion',
    trigger_value TEXT NOT NULL DEFAULT '',
    test_event_code TEXT NOT NULL DEFAULT '',
    signing_secret_encrypted BYTEA NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_campaign_outbound_postbacks_campaign
    ON campaign_outbound_postbacks(campaign_id);

CREATE INDEX IF NOT EXISTS idx_campaign_outbound_postbacks_campaign_priority
    ON campaign_outbound_postbacks(campaign_id, priority ASC);

-- +goose Down
DROP TABLE IF EXISTS campaign_outbound_postbacks;
