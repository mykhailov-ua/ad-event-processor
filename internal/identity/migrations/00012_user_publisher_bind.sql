-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS seller_id TEXT,
    ADD COLUMN IF NOT EXISTS publisher_account_id TEXT;

CREATE INDEX IF NOT EXISTS idx_users_seller_id ON users (seller_id) WHERE seller_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_users_seller_id;
ALTER TABLE users
    DROP COLUMN IF EXISTS publisher_account_id,
    DROP COLUMN IF EXISTS seller_id;
-- +goose StatementEnd
