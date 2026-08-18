-- +goose Up
-- +goose StatementBegin
ALTER TABLE ml_manual_labels
    ADD COLUMN IF NOT EXISTS customer_id UUID NULL REFERENCES customers(id);

CREATE INDEX IF NOT EXISTS idx_ml_manual_labels_customer_created
    ON ml_manual_labels (customer_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_ml_manual_labels_customer_created;
ALTER TABLE ml_manual_labels DROP COLUMN IF EXISTS customer_id;
-- +goose StatementEnd
