-- Index for time-bounded post-failover duplicate audit (GAP-GEO-02).
-- Supports WHERE created_at >= $since GROUP BY idempotency_hash on recent rows only.

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
