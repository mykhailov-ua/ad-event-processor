-- +goose Up
-- +goose StatementBegin
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS target_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS creative_payload JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS referrer_filter TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE campaigns
    DROP COLUMN IF EXISTS target_url,
    DROP COLUMN IF EXISTS creative_payload,
    DROP COLUMN IF EXISTS referrer_filter;
-- +goose StatementEnd
