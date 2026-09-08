-- +goose Up
-- +goose StatementBegin
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS cross_layer_desync_action TEXT NOT NULL DEFAULT 'boost',
    ADD COLUMN IF NOT EXISTS cross_layer_desync_threshold SMALLINT NOT NULL DEFAULT 3;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE campaigns
    DROP COLUMN IF EXISTS cross_layer_desync_action,
    DROP COLUMN IF EXISTS cross_layer_desync_threshold;
-- +goose StatementEnd
