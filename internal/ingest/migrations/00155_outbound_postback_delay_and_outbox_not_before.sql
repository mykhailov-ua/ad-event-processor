-- +goose Up
ALTER TABLE campaign_outbound_postbacks
    ADD COLUMN IF NOT EXISTS delay_seconds INT NOT NULL DEFAULT 0;

ALTER TABLE outbox_events
    ADD COLUMN IF NOT EXISTS not_before TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_outbox_events_postback_not_before
    ON outbox_events (not_before)
    WHERE event_type = 'SEND_POSTBACK' AND status = 'PENDING';

-- +goose Down
DROP INDEX IF EXISTS idx_outbox_events_postback_not_before;

ALTER TABLE outbox_events
    DROP COLUMN IF EXISTS not_before;

ALTER TABLE campaign_outbound_postbacks
    DROP COLUMN IF EXISTS delay_seconds;
