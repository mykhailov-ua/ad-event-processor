-- +goose Up
-- +goose StatementBegin
ALTER TABLE sync_idempotency
    ADD COLUMN IF NOT EXISTS event_id TEXT,
    ADD COLUMN IF NOT EXISTS campaign_id UUID;

CREATE UNIQUE INDEX IF NOT EXISTS sync_idempotency_event_campaign_uidx
    ON sync_idempotency (event_id, campaign_id)
    WHERE event_id IS NOT NULL AND campaign_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS sync_idempotency_event_campaign_uidx;
ALTER TABLE sync_idempotency DROP COLUMN IF EXISTS campaign_id;
ALTER TABLE sync_idempotency DROP COLUMN IF EXISTS event_id;
-- +goose StatementEnd
