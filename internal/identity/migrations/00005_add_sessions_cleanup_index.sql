-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_sessions_cleanup ON sessions(expires_at, is_blocked);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF NOT EXISTS idx_sessions_cleanup;
-- +goose StatementEnd
