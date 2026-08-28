-- +goose Up
-- +goose StatementBegin
ALTER TABLE report_saved_views
    ADD COLUMN IF NOT EXISTS owner_mask_level TEXT NOT NULL DEFAULT 'masked';

ALTER TABLE report_saved_views
    DROP CONSTRAINT IF EXISTS report_saved_views_owner_mask_level_check;

ALTER TABLE report_saved_views
    ADD CONSTRAINT report_saved_views_owner_mask_level_check
    CHECK (owner_mask_level IN ('full', 'masked'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE report_saved_views DROP CONSTRAINT IF EXISTS report_saved_views_owner_mask_level_check;
ALTER TABLE report_saved_views DROP COLUMN IF EXISTS owner_mask_level;
-- +goose StatementEnd
