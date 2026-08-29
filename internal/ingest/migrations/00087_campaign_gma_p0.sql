-- +goose Up
-- +goose StatementBegin
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS tls_fingerprint_block_enabled BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS conn_type_policy TEXT NOT NULL DEFAULT 'block_vpn_hosting'
        CHECK (conn_type_policy IN ('block_vpn_hosting', 'mobile_only', 'residential_only')),
    ADD COLUMN IF NOT EXISTS link_signing_enabled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS link_signing_ttl_sec INT NOT NULL DEFAULT 900
        CHECK (link_signing_ttl_sec BETWEEN 60 AND 3600);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE campaigns
    DROP COLUMN IF EXISTS link_signing_ttl_sec,
    DROP COLUMN IF EXISTS link_signing_enabled,
    DROP COLUMN IF EXISTS conn_type_policy,
    DROP COLUMN IF EXISTS tls_fingerprint_block_enabled;
-- +goose StatementEnd
