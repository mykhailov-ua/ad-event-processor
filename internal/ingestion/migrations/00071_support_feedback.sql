-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS support;

CREATE TABLE support.feedback (
    id              UUID PRIMARY KEY,
    feedback_type   TEXT NOT NULL CHECK (feedback_type IN ('bug', 'feature', 'support')),
    contact_email   TEXT NOT NULL,
    message         TEXT NOT NULL CHECK (char_length(message) >= 1 AND char_length(message) <= 8000),
    deployment_id   TEXT NOT NULL DEFAULT '',
    binary_version  TEXT NOT NULL DEFAULT '',
    sku             TEXT NOT NULL DEFAULT '',
    attach_bundle   BOOLEAN NOT NULL DEFAULT FALSE,
    bundle_gzip     BYTEA,
    submitter_id    UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_support_feedback_created_at ON support.feedback (created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS support.feedback;
DROP SCHEMA IF EXISTS support;
-- +goose StatementEnd
