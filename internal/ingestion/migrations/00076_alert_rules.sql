-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS alert_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    campaign_id UUID REFERENCES campaigns(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    metric TEXT NOT NULL CHECK (metric IN ('clicks', 'cr', 'roi_pct', 'bot_clicks')),
    operator TEXT NOT NULL CHECK (operator IN ('gt', 'lt', 'gte', 'lte')),
    threshold DOUBLE PRECISION NOT NULL,
    window_minutes INT NOT NULL CHECK (window_minutes BETWEEN 5 AND 1440),
    webhook_url TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_alert_rules_customer ON alert_rules(customer_id);
CREATE INDEX IF NOT EXISTS idx_alert_rules_enabled ON alert_rules(enabled) WHERE enabled = true;

CREATE TABLE IF NOT EXISTS alert_rule_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID NOT NULL REFERENCES alert_rules(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL,
    campaign_id UUID,
    window_start TIMESTAMPTZ NOT NULL,
    window_end TIMESTAMPTZ NOT NULL,
    metric TEXT NOT NULL,
    operator TEXT NOT NULL,
    threshold DOUBLE PRECISION NOT NULL,
    observed_value DOUBLE PRECISION NOT NULL,
    webhook_status TEXT NOT NULL DEFAULT 'pending',
    webhook_error TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL DEFAULT '{}',
    fired_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    acked_at TIMESTAMPTZ,
    acked_by UUID,
    UNIQUE (rule_id, window_start)
);

CREATE INDEX IF NOT EXISTS idx_alert_rule_events_customer ON alert_rule_events(customer_id, fired_at DESC);
CREATE INDEX IF NOT EXISTS idx_alert_rule_events_rule ON alert_rule_events(rule_id, fired_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS alert_rule_events;
DROP TABLE IF EXISTS alert_rules;
-- +goose StatementEnd
