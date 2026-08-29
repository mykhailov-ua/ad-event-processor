-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS integration_schemas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    version INT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('inbound_tokens', 'outbound_postback', 'status_mapping')),
    body JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (name, version)
);

CREATE INDEX IF NOT EXISTS idx_integration_schemas_kind ON integration_schemas(kind);

ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS integration_schema_id UUID REFERENCES integration_schemas(id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE campaigns DROP COLUMN IF EXISTS integration_schema_id;
DROP TABLE IF EXISTS integration_schemas;
-- +goose StatementEnd
