-- +goose Up
-- +goose StatementBegin
ALTER TYPE ledger_type ADD VALUE IF NOT EXISTS 'publisher_payout';
ALTER TYPE ledger_type ADD VALUE IF NOT EXISTS 'operator_margin';
ALTER TYPE ledger_type ADD VALUE IF NOT EXISTS 'rtb_cost';

CREATE INDEX IF NOT EXISTS idx_balance_ledger_campaign_margin
  ON balance_ledger (campaign_id, created_at DESC)
  WHERE type IN ('FEE', 'rtb_cost', 'operator_margin', 'publisher_payout');

ALTER TABLE margin_guard_policies
  ADD COLUMN IF NOT EXISTS cost_over_revenue_threshold_bps INT NOT NULL DEFAULT 500;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE margin_guard_policies DROP COLUMN IF EXISTS cost_over_revenue_threshold_bps;
DROP INDEX IF EXISTS idx_balance_ledger_campaign_margin;
-- +goose StatementEnd
