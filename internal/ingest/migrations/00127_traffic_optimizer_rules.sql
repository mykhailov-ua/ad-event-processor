-- +goose Up
CREATE TABLE IF NOT EXISTS traffic_optimizer_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    campaign_id UUID REFERENCES campaigns(id) ON DELETE CASCADE,
    flow_id UUID REFERENCES flows(id) ON DELETE CASCADE,
    brand_id UUID REFERENCES advertiser_brands(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    scope TEXT NOT NULL CHECK (scope IN ('lander', 'offer', 'creative')),
    objective TEXT NOT NULL CHECK (objective IN ('cr', 'epc', 'roi', 'revenue')),
    algorithm TEXT NOT NULL DEFAULT 'proportional' CHECK (algorithm IN ('thompson', 'proportional')),
    lookback_minutes INT NOT NULL DEFAULT 1440,
    min_clicks INT NOT NULL DEFAULT 100,
    min_conversions INT NOT NULL DEFAULT 0,
    min_spend_micro BIGINT NOT NULL DEFAULT 0,
    eval_interval_minutes INT NOT NULL DEFAULT 15,
    cooldown_minutes INT NOT NULL DEFAULT 60,
    max_weight_delta_pct INT NOT NULL DEFAULT 50,
    preset_key TEXT,
    preset_parameters JSONB NOT NULL DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT true,
    last_evaluated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_traffic_optimizer_rules_customer ON traffic_optimizer_rules(customer_id);
CREATE INDEX IF NOT EXISTS idx_traffic_optimizer_rules_customer_enabled ON traffic_optimizer_rules(customer_id, enabled);

CREATE TABLE IF NOT EXISTS traffic_optimizer_fires (
    rule_id UUID NOT NULL REFERENCES traffic_optimizer_rules(id) ON DELETE CASCADE,
    action_hash TEXT NOT NULL,
    campaign_id UUID,
    flow_id UUID,
    brand_id UUID,
    payload JSONB NOT NULL DEFAULT '{}',
    fired_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (rule_id, action_hash)
);

CREATE INDEX IF NOT EXISTS idx_traffic_optimizer_fires_campaign ON traffic_optimizer_fires(campaign_id, fired_at DESC);

-- +goose Down
DROP TABLE IF EXISTS traffic_optimizer_fires;
DROP TABLE IF EXISTS traffic_optimizer_rules;
