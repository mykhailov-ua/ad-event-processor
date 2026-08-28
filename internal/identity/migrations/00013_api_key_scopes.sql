-- +goose Up
-- +goose StatementBegin
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS scopes TEXT[] NOT NULL DEFAULT '{}';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE api_keys DROP COLUMN IF EXISTS scopes;
-- +goose StatementEnd
