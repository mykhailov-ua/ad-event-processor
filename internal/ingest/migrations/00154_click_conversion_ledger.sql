-- +goose Up
CREATE TABLE IF NOT EXISTS click_conversion_ledger (
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    click_id TEXT NOT NULL,
    payout_micro BIGINT NOT NULL DEFAULT 0,
    last_status TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (campaign_id, click_id)
);

CREATE INDEX IF NOT EXISTS idx_click_conversion_ledger_updated_at
    ON click_conversion_ledger (updated_at DESC);

-- +goose Down
DROP TABLE IF EXISTS click_conversion_ledger;
