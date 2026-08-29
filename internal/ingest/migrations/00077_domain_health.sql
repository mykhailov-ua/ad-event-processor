-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS domain_health_status (
    hostname TEXT PRIMARY KEY,
    role TEXT NOT NULL CHECK (role IN ('tracking', 'admin', 'custom')),
    health_status TEXT NOT NULL DEFAULT 'unknown'
        CHECK (health_status IN ('healthy', 'degraded', 'down', 'unknown')),
    ssl_status TEXT NOT NULL DEFAULT 'unknown'
        CHECK (ssl_status IN ('valid', 'expiring', 'expired', 'missing', 'unknown')),
    ssl_not_after TIMESTAMPTZ,
    http_status INT,
    probe_latency_ms INT,
    probe_detail TEXT NOT NULL DEFAULT '',
    last_probe_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_domain_health_role ON domain_health_status(role);
CREATE INDEX IF NOT EXISTS idx_domain_health_last_probe ON domain_health_status(last_probe_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS domain_health_status;
-- +goose StatementEnd
