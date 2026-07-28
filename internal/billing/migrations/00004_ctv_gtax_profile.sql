-- +goose Up
-- +goose StatementBegin
ALTER TABLE billing.customer_tax_profiles
  ADD COLUMN IF NOT EXISTS ctv_gtax_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS ctv_gtax_rate_bps INT NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE billing.customer_tax_profiles
  DROP COLUMN IF EXISTS ctv_gtax_rate_bps,
  DROP COLUMN IF EXISTS ctv_gtax_enabled;
-- +goose StatementEnd
