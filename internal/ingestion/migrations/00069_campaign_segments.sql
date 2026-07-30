-- +goose Up
-- +goose StatementBegin
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS retarget_segment_id UUID,
    ADD COLUMN IF NOT EXISTS segment_ttl_hours INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS segment_include UUID,
    ADD COLUMN IF NOT EXISTS segment_exclude UUID;

CREATE TABLE IF NOT EXISTS segment_members (
    segment_id UUID NOT NULL,
    user_hash BYTEA NOT NULL CHECK (octet_length(user_hash) = 16),
    added_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ,
    PRIMARY KEY (segment_id, user_hash)
);

CREATE INDEX IF NOT EXISTS idx_segment_members_expires
    ON segment_members (expires_at)
    WHERE expires_at IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS segment_members;

ALTER TABLE campaigns
    DROP COLUMN IF EXISTS retarget_segment_id,
    DROP COLUMN IF EXISTS segment_ttl_hours,
    DROP COLUMN IF EXISTS segment_include,
    DROP COLUMN IF EXISTS segment_exclude;
-- +goose StatementEnd
