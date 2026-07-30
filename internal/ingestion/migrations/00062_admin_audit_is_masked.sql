-- +goose Up
-- +goose StatementBegin
ALTER TABLE admin_audit_log
    ADD COLUMN IF NOT EXISTS is_masked BOOLEAN NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE admin_audit_log DROP COLUMN IF EXISTS is_masked;
-- +goose StatementEnd
