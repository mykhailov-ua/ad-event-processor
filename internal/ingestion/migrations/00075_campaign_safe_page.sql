-- +goose Up
-- +goose StatementBegin
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS safe_page_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS safe_page_enabled BOOLEAN NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE campaigns
    DROP COLUMN IF EXISTS safe_page_enabled,
    DROP COLUMN IF EXISTS safe_page_url;
-- +goose StatementEnd
