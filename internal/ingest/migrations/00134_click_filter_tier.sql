-- +goose Up
-- +goose StatementBegin
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS click_filter_tier TEXT NOT NULL DEFAULT 'full';

ALTER TABLE campaigns
    DROP CONSTRAINT IF EXISTS campaigns_click_filter_tier_check;

ALTER TABLE campaigns
    ADD CONSTRAINT campaigns_click_filter_tier_check
    CHECK (click_filter_tier IN ('full', 'light', 'redirect_only'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE campaigns DROP CONSTRAINT IF EXISTS campaigns_click_filter_tier_check;
ALTER TABLE campaigns DROP COLUMN IF EXISTS click_filter_tier;
-- +goose StatementEnd
