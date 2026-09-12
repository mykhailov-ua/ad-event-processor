-- +goose Up
ALTER TABLE alert_rules
    DROP CONSTRAINT IF EXISTS alert_rules_metric_check;

ALTER TABLE alert_rules
    ADD COLUMN IF NOT EXISTS action TEXT NOT NULL DEFAULT 'notify';

ALTER TABLE alert_rules
    DROP CONSTRAINT IF EXISTS alert_rules_action_check;

ALTER TABLE alert_rules
    ADD CONSTRAINT alert_rules_action_check
    CHECK (action IN ('notify', 'pause_campaign', 'blacklist_placement'));

-- +goose Down
ALTER TABLE alert_rules
    DROP CONSTRAINT IF EXISTS alert_rules_action_check;

ALTER TABLE alert_rules
    DROP COLUMN IF EXISTS action;

ALTER TABLE alert_rules
    DROP CONSTRAINT IF EXISTS alert_rules_metric_check;

ALTER TABLE alert_rules
    ADD CONSTRAINT alert_rules_metric_check
    CHECK (metric IN ('clicks', 'cr', 'roi_pct', 'bot_clicks'));
