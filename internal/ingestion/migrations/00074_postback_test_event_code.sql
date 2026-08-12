-- +goose Up
-- +goose StatementBegin
ALTER TABLE postback_configs
    ADD COLUMN IF NOT EXISTS test_event_code TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE postback_configs DROP COLUMN IF EXISTS test_event_code;
-- +goose StatementEnd
