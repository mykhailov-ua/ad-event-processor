-- +goose Up
-- +goose StatementBegin
ALTER TABLE events
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'accepted'
    CHECK (status IN ('accepted', 'rejected'));

CREATE INDEX IF NOT EXISTS events_created_at_status_idx
    ON events (created_at)
    WHERE status = 'accepted';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS events_created_at_status_idx;
ALTER TABLE events DROP COLUMN IF EXISTS status;
-- +goose StatementEnd
