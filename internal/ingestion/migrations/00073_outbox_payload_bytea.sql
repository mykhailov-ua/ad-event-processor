-- +goose Up
-- +goose StatementBegin
-- Protobuf payloads are opaque bytes; legacy JSON rows remain valid UTF-8 JSON.
ALTER TABLE outbox_events
  ALTER COLUMN payload TYPE BYTEA USING convert_to(payload::text, 'UTF8');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE outbox_events
  ALTER COLUMN payload TYPE JSONB USING convert_from(payload, 'UTF8')::jsonb;
-- +goose StatementEnd
