-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS domain_wildcard_ssl (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pool_id UUID NOT NULL REFERENCES domain_pools (id) ON DELETE CASCADE,
    cloudflare_zone_id TEXT NOT NULL,
    zone_name TEXT NOT NULL,
    wildcard_hostname TEXT NOT NULL,
    include_apex BOOLEAN NOT NULL DEFAULT false,
    acme_state TEXT NOT NULL DEFAULT 'pending'
        CHECK (acme_state IN ('pending', 'valid', 'failed', 'renewing')),
    cloudflare_proxied BOOLEAN NOT NULL DEFAULT true,
    cert_pem TEXT NOT NULL DEFAULT '',
    key_pem_encrypted BYTEA NOT NULL DEFAULT ''::bytea,
    ssl_not_after TIMESTAMPTZ,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (pool_id, zone_name)
);

CREATE INDEX IF NOT EXISTS idx_domain_wildcard_ssl_state ON domain_wildcard_ssl (acme_state, ssl_not_after);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS domain_wildcard_ssl;
-- +goose StatementEnd
