-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_recon_disc_customer_created
    ON recon_discrepancies (customer_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_recon_disc_customer_created;
-- +goose StatementEnd
