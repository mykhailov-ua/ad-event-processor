-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS ml_manual_labels (
    ip_hash TEXT PRIMARY KEY,
    label INT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT 'admin_ui',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS ml_manual_labels;
-- +goose StatementEnd
