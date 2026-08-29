-- +goose Up
-- +goose StatementBegin
ALTER TABLE integration_schemas DROP CONSTRAINT IF EXISTS integration_schemas_kind_check;
ALTER TABLE integration_schemas ADD CONSTRAINT integration_schemas_kind_check
    CHECK (kind IN ('inbound_tokens', 'outbound_postback', 'affiliate_receive_postback', 'status_mapping'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE integration_schemas DROP CONSTRAINT IF EXISTS integration_schemas_kind_check;
ALTER TABLE integration_schemas ADD CONSTRAINT integration_schemas_kind_check
    CHECK (kind IN ('inbound_tokens', 'outbound_postback', 'status_mapping'));
-- +goose StatementEnd
