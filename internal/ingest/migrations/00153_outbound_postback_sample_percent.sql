-- +goose Up
ALTER TABLE campaign_outbound_postbacks
    ADD COLUMN IF NOT EXISTS sample_percent INT NOT NULL DEFAULT 100;

-- +goose Down
ALTER TABLE campaign_outbound_postbacks
    DROP COLUMN IF EXISTS sample_percent;
