-- +goose Up
-- +goose StatementBegin

ALTER TABLE vendor.licenses
    ADD COLUMN IF NOT EXISTS max_activations INT NOT NULL DEFAULT 1;

CREATE TABLE vendor.license_activations (
    license_key    TEXT NOT NULL REFERENCES vendor.licenses(license_key) ON DELETE CASCADE,
    fingerprint    TEXT NOT NULL,
    deployment_id  UUID NOT NULL,
    first_seen_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (license_key, fingerprint)
);

CREATE INDEX idx_license_activations_deployment ON vendor.license_activations (deployment_id);

CREATE TABLE vendor.license_revoke_queue (
    id             BIGSERIAL PRIMARY KEY,
    license_key    TEXT NOT NULL,
    reason         TEXT NOT NULL,
    detail_json    JSONB,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at   TIMESTAMPTZ
);

CREATE INDEX idx_license_revoke_queue_pending ON vendor.license_revoke_queue (license_key)
    WHERE processed_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS vendor.license_revoke_queue;
DROP TABLE IF EXISTS vendor.license_activations;
ALTER TABLE vendor.licenses DROP COLUMN IF EXISTS max_activations;

-- +goose StatementEnd
