-- +goose Up
-- +goose StatementBegin

ALTER TABLE customers ADD COLUMN IF NOT EXISTS cost_center TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_customers_cost_center ON customers (cost_center) WHERE cost_center <> '';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_customers_cost_center;
ALTER TABLE customers DROP COLUMN IF EXISTS cost_center;

-- +goose StatementEnd
