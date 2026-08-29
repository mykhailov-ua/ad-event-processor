-- +goose Up
-- +goose StatementBegin
CREATE TABLE telegram_bots (
    campaign_id UUID PRIMARY KEY REFERENCES campaigns(id) ON DELETE CASCADE,
    bot_id BIGINT UNIQUE NOT NULL,
    bot_token TEXT NOT NULL,
    webhook_url TEXT NOT NULL,
    secret_token TEXT NOT NULL,
    auth_date_ttl INT NOT NULL DEFAULT 300,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE telegram_deeplinks (
    token VARCHAR(64) PRIMARY KEY,
    campaign_id UUID REFERENCES campaigns(id) ON DELETE CASCADE,
    fbclid TEXT,
    ttclid TEXT,
    utm_source TEXT,
    utm_medium TEXT,
    utm_campaign TEXT,
    utm_term TEXT,
    utm_content TEXT,
    landing_ts TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE telegram_webhook_idempotency (
    update_id BIGINT PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE telegram_postbacks (
    id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    postback_url TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS telegram_postbacks;
DROP TABLE IF EXISTS telegram_webhook_idempotency;
DROP TABLE IF EXISTS telegram_deeplinks;
DROP TABLE IF EXISTS telegram_bots;
-- +goose StatementEnd
