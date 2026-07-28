-- +goose Up
-- +goose StatementBegin
ALTER TYPE ledger_type ADD VALUE IF NOT EXISTS 'CTV_GTAX';

CREATE TABLE ctv_gtax_settlements (
  settlement_id   TEXT PRIMARY KEY,
  customer_id     UUID NOT NULL REFERENCES customers(id),
  campaign_id     UUID NOT NULL REFERENCES campaigns(id),
  spend_micro     BIGINT NOT NULL,
  tax_micro       BIGINT NOT NULL,
  fee_ledger_id   BIGINT NOT NULL REFERENCES balance_ledger(id),
  tax_ledger_id   BIGINT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_ctv_gtax_settlements_campaign
  ON ctv_gtax_settlements (campaign_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS ctv_gtax_settlements;
-- Note: PostgreSQL cannot remove enum values; CTV_GTAX remains on ledger_type.
-- +goose StatementEnd
