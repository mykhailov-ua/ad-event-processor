-- +goose Up
-- +goose StatementBegin
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS status_integration_schema_id UUID REFERENCES integration_schemas (id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE campaigns DROP COLUMN IF EXISTS status_integration_schema_id;
-- +goose StatementEnd
