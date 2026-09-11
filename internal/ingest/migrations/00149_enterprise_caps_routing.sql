-- +goose Up
-- +goose StatementBegin
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS fallback_click_url TEXT;

ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS budget_failover_mode TEXT NOT NULL DEFAULT 'none';

ALTER TABLE campaigns
    DROP CONSTRAINT IF EXISTS campaigns_budget_failover_mode_check;

ALTER TABLE campaigns
    ADD CONSTRAINT campaigns_budget_failover_mode_check
    CHECK (budget_failover_mode IN ('none', 'fallback_url', 'flow_next'));

ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS click_filter_budget_policy TEXT NOT NULL DEFAULT 'full';

ALTER TABLE campaigns
    DROP CONSTRAINT IF EXISTS campaigns_click_filter_budget_policy_check;

ALTER TABLE campaigns
    ADD CONSTRAINT campaigns_click_filter_budget_policy_check
    CHECK (click_filter_budget_policy IN ('inherit', 'full', 'light_skip_debit', 'redirect_skip_debit'));

ALTER TABLE flows
    ADD COLUMN IF NOT EXISTS flow_routing_mode TEXT NOT NULL DEFAULT 'weighted';

ALTER TABLE flows
    DROP CONSTRAINT IF EXISTS flows_flow_routing_mode_check;

ALTER TABLE flows
    ADD CONSTRAINT flows_flow_routing_mode_check
    CHECK (flow_routing_mode IN ('weighted', 'waterfall'));

ALTER TABLE offers
    ADD COLUMN IF NOT EXISTS offer_priority INT NOT NULL DEFAULT 0;

ALTER TABLE offers
    ADD COLUMN IF NOT EXISTS cap_clicks_daily INT NOT NULL DEFAULT 0;

ALTER TABLE offers
    ADD COLUMN IF NOT EXISTS cap_clicks_total INT NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE offers DROP COLUMN IF EXISTS cap_clicks_total;
ALTER TABLE offers DROP COLUMN IF EXISTS cap_clicks_daily;
ALTER TABLE offers DROP COLUMN IF EXISTS offer_priority;

ALTER TABLE flows DROP CONSTRAINT IF EXISTS flows_flow_routing_mode_check;
ALTER TABLE flows DROP COLUMN IF EXISTS flow_routing_mode;

ALTER TABLE campaigns DROP CONSTRAINT IF EXISTS campaigns_click_filter_budget_policy_check;
ALTER TABLE campaigns DROP COLUMN IF EXISTS click_filter_budget_policy;

ALTER TABLE campaigns DROP CONSTRAINT IF EXISTS campaigns_budget_failover_mode_check;
ALTER TABLE campaigns DROP COLUMN IF EXISTS budget_failover_mode;

ALTER TABLE campaigns DROP COLUMN IF EXISTS fallback_click_url;
-- +goose StatementEnd
