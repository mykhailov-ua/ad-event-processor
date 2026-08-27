-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS platform_campaign_links (
    id BIGSERIAL PRIMARY KEY,
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL REFERENCES customers(id),
    network TEXT NOT NULL,
    external_campaign_id TEXT NOT NULL,
    account_id TEXT NOT NULL DEFAULT '',
    external_status TEXT NOT NULL DEFAULT '',
    external_budget_resource TEXT NOT NULL DEFAULT '',
    external_daily_budget_micro BIGINT,
    last_synced_at TIMESTAMPTZ,
    sync_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (campaign_id, network)
);

CREATE INDEX idx_platform_campaign_links_customer ON platform_campaign_links(customer_id);
CREATE INDEX idx_platform_campaign_links_sync ON platform_campaign_links(last_synced_at);

CREATE TABLE IF NOT EXISTS platform_campaign_mutations (
    id BIGSERIAL PRIMARY KEY,
    idempotency_key TEXT NOT NULL UNIQUE,
    campaign_id UUID NOT NULL REFERENCES campaigns(id),
    customer_id UUID NOT NULL REFERENCES customers(id),
    network TEXT NOT NULL,
    action TEXT NOT NULL,
    request_json JSONB NOT NULL DEFAULT '{}',
    response_json JSONB,
    status TEXT NOT NULL DEFAULT 'pending',
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    applied_at TIMESTAMPTZ
);

CREATE INDEX idx_platform_campaign_mutations_pending
    ON platform_campaign_mutations(status, created_at)
    WHERE status = 'pending';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS platform_campaign_mutations;
DROP TABLE IF EXISTS platform_campaign_links;
-- +goose StatementEnd
