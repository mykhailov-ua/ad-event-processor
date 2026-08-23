-- +goose Up
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS attestation_mode TEXT NOT NULL DEFAULT 'off'
        CHECK (attestation_mode IN ('off', 'light', 'strict'));

UPDATE campaigns
SET attestation_mode = 'strict'
WHERE attestation_enabled = true AND attestation_mode = 'off';

-- +goose Down
ALTER TABLE campaigns
    DROP COLUMN IF EXISTS attestation_mode;
