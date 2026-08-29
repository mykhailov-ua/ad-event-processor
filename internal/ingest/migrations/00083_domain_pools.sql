-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS domain_pools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_domain_pools_name ON domain_pools (name);

CREATE TABLE IF NOT EXISTS domain_pool_domains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pool_id UUID NOT NULL REFERENCES domain_pools (id) ON DELETE CASCADE,
    hostname TEXT NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'active', 'banned')),
    cloudflare_zone_id TEXT NOT NULL DEFAULT '',
    dns_record_id TEXT NOT NULL DEFAULT '',
    ssl_status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (pool_id, hostname)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_domain_pool_domains_hostname ON domain_pool_domains (hostname);
CREATE INDEX IF NOT EXISTS idx_domain_pool_domains_pool_status ON domain_pool_domains (pool_id, status, sort_order);

ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS domain_pool_id UUID REFERENCES domain_pools (id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE campaigns DROP COLUMN IF EXISTS domain_pool_id;
DROP TABLE IF EXISTS domain_pool_domains;
DROP TABLE IF EXISTS domain_pools;
-- +goose StatementEnd
