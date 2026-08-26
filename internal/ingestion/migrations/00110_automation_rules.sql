-- +goose Up
CREATE TABLE IF NOT EXISTS automation_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    campaign_id UUID REFERENCES campaigns(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    metric TEXT NOT NULL,
    operator TEXT NOT NULL,
    threshold DOUBLE PRECISION NOT NULL,
    window_minutes INT NOT NULL DEFAULT 60,
    group_by TEXT NOT NULL DEFAULT 'placement_id',
    actions JSONB NOT NULL DEFAULT '[]',
    cooldown_minutes INT NOT NULL DEFAULT 60,
    enabled BOOLEAN NOT NULL DEFAULT true,
    last_fired_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_automation_rules_customer ON automation_rules(customer_id);
CREATE INDEX IF NOT EXISTS idx_automation_rules_enabled ON automation_rules(enabled) WHERE enabled = true;

CREATE TABLE IF NOT EXISTS automation_rule_fires (
    rule_id UUID NOT NULL REFERENCES automation_rules(id) ON DELETE CASCADE,
    action_hash TEXT NOT NULL,
    campaign_id UUID NOT NULL,
    placement_id TEXT NOT NULL DEFAULT '',
    metric TEXT NOT NULL,
    observed_value DOUBLE PRECISION NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    fired_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (rule_id, action_hash)
);

CREATE INDEX IF NOT EXISTS idx_automation_rule_fires_campaign ON automation_rule_fires(campaign_id, fired_at DESC);

-- +goose Down
DROP TABLE IF EXISTS automation_rule_fires;
DROP TABLE IF EXISTS automation_rules;
