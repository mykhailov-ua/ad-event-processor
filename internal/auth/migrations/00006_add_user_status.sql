-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN is_blocked BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX idx_users_is_blocked ON users(id, is_blocked);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_users_is_blocked;
ALTER TABLE users DROP COLUMN IF EXISTS is_blocked;
-- +goose StatementEnd
