-- +goose Up
-- +goose StatementBegin
ALTER TABLE postback_configs
    ADD COLUMN IF NOT EXISTS signing_secret_encrypted BYTEA;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE postback_configs DROP COLUMN IF EXISTS signing_secret_encrypted;
-- +goose StatementEnd
