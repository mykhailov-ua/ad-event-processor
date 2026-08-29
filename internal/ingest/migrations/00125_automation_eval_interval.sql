-- +goose Up
ALTER TABLE automation_rules
    ADD COLUMN IF NOT EXISTS eval_interval_minutes INT NOT NULL DEFAULT 15,
    ADD COLUMN IF NOT EXISTS last_evaluated_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE automation_rules
    DROP COLUMN IF EXISTS last_evaluated_at,
    DROP COLUMN IF EXISTS eval_interval_minutes;
