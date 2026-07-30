-- +goose Up
-- +goose StatementBegin
ALTER TYPE ledger_type ADD VALUE IF NOT EXISTS 'publisher_payout';
ALTER TYPE ledger_type ADD VALUE IF NOT EXISTS 'operator_margin';
ALTER TYPE ledger_type ADD VALUE IF NOT EXISTS 'rtb_cost';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- PostgreSQL cannot remove enum values; no-op down migration.
-- +goose StatementEnd
