-- +goose Up
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS attestation_enabled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS attestation_ttl_sec INT NOT NULL DEFAULT 300
        CHECK (attestation_ttl_sec BETWEEN 60 AND 900);

-- +goose Down
ALTER TABLE campaigns
    DROP COLUMN IF EXISTS attestation_ttl_sec,
    DROP COLUMN IF EXISTS attestation_enabled;
