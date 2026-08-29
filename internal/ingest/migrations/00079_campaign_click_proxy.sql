-- +goose Up
-- +goose StatementBegin
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS click_delivery TEXT NOT NULL DEFAULT 'redirect'
        CHECK (click_delivery IN ('redirect', 'proxy')),
    ADD COLUMN IF NOT EXISTS proxy_upstream_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS proxy_rewrite_assets BOOLEAN NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE campaigns
    DROP COLUMN IF EXISTS proxy_rewrite_assets,
    DROP COLUMN IF EXISTS proxy_upstream_url,
    DROP COLUMN IF EXISTS click_delivery;
-- +goose StatementEnd
