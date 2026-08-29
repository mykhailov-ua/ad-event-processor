-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_admin_audit_log_created_at_desc
    ON admin_audit_log (created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_admin_audit_log_created_at_desc;
-- +goose StatementEnd
