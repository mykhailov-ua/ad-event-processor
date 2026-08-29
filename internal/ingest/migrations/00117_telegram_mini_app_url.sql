-- +goose Up
-- +goose StatementBegin
ALTER TABLE telegram_bots
    ADD COLUMN IF NOT EXISTS mini_app_url TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE telegram_bots DROP COLUMN IF EXISTS mini_app_url;
-- +goose StatementEnd
