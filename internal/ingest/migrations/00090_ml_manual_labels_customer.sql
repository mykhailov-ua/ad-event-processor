-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS ml_manual_labels (
    ip_hash TEXT PRIMARY KEY,
    label INT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT 'admin_ui',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE ml_manual_labels
    ADD COLUMN IF NOT EXISTS customer_id UUID NULL REFERENCES customers(id);

CREATE INDEX IF NOT EXISTS idx_ml_manual_labels_customer_created
    ON ml_manual_labels (customer_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_ml_manual_labels_customer_created;
ALTER TABLE ml_manual_labels DROP COLUMN IF EXISTS customer_id;
DROP TABLE IF EXISTS ml_manual_labels;
-- +goose StatementEnd
