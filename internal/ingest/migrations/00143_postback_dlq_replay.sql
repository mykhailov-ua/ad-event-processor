-- +goose Up
-- +goose StatementBegin
ALTER TABLE postback_dlq
    ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS replay_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_replay_count INT NOT NULL DEFAULT 5;

CREATE INDEX IF NOT EXISTS idx_postback_dlq_auto_replay
    ON postback_dlq(status, next_retry_at)
    WHERE status = 'FAILED';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_postback_dlq_auto_replay;
ALTER TABLE postback_dlq
    DROP COLUMN IF EXISTS max_replay_count,
    DROP COLUMN IF EXISTS replay_count,
    DROP COLUMN IF EXISTS next_retry_at;
-- +goose StatementEnd
