-- +goose Up
-- +goose StatementBegin

ALTER TYPE ledger_type ADD VALUE IF NOT EXISTS 'RECONCILIATION_ADJUST';

CREATE TABLE IF NOT EXISTS recon_runs (
    id BIGSERIAL PRIMARY KEY,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING',
    total_delta BIGINT NOT NULL DEFAULT 0,
    campaigns_checked INT NOT NULL DEFAULT 0,
    discrepancies_found INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_recon_runs_period ON recon_runs (period_start, period_end);
CREATE INDEX IF NOT EXISTS idx_recon_runs_status ON recon_runs (status);

CREATE TABLE IF NOT EXISTS recon_discrepancies (
    id BIGSERIAL PRIMARY KEY,
    run_id BIGINT NOT NULL REFERENCES recon_runs(id) ON DELETE CASCADE,
    campaign_id UUID NOT NULL,
    customer_id UUID NOT NULL,
    expected_spend BIGINT NOT NULL,
    actual_spend BIGINT NOT NULL,
    delta BIGINT NOT NULL,
    adjustment_ledger_id BIGINT,
    redis_adjusted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recon_disc_run ON recon_discrepancies (run_id);
CREATE INDEX IF NOT EXISTS idx_recon_disc_campaign ON recon_discrepancies (campaign_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS recon_discrepancies;
DROP TABLE IF EXISTS recon_runs;
-- +goose StatementEnd
