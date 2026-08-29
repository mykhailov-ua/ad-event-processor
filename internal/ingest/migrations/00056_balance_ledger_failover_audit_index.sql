-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_ledger_failover_audit
    ON balance_ledger (created_at DESC, idempotency_hash)
    WHERE idempotency_hash IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_ledger_failover_audit;
-- +goose StatementEnd
