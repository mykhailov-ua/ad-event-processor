-- +goose Up
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS review_traffic_action TEXT NOT NULL DEFAULT 'safe_page';

-- +goose Down
ALTER TABLE campaigns
    DROP COLUMN IF EXISTS review_traffic_action;
