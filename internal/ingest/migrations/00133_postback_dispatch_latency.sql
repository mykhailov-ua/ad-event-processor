-- +goose Up
-- +goose StatementBegin
ALTER TABLE postback_dispatches
    ADD COLUMN IF NOT EXISTS latency_ms INT;

CREATE INDEX IF NOT EXISTS idx_postback_dispatches_campaign_created
    ON postback_dispatches (campaign_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_postback_dispatches_campaign_created;
ALTER TABLE postback_dispatches DROP COLUMN IF EXISTS latency_ms;
-- +goose StatementEnd
