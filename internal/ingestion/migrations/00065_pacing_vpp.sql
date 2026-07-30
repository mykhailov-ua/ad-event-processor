-- +goose Up
-- +goose StatementBegin
ALTER TYPE pacing_mode_type ADD VALUE IF NOT EXISTS 'VPP';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- PostgreSQL does not support removing enum values; no-op down migration.
-- +goose StatementEnd
