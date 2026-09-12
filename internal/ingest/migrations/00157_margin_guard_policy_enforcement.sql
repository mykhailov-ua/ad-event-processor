-- +goose Up
ALTER TABLE margin_guard_policies
    ADD COLUMN IF NOT EXISTS enforcement TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS cooldown_sec INT NOT NULL DEFAULT 3600,
    ADD COLUMN IF NOT EXISTS platform_pause BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS platform_network TEXT NOT NULL DEFAULT '';

ALTER TABLE margin_guard_policies
    DROP CONSTRAINT IF EXISTS margin_guard_policies_enforcement_check;

ALTER TABLE margin_guard_policies
    ADD CONSTRAINT margin_guard_policies_enforcement_check
    CHECK (enforcement IN ('', 'pause_campaign', 'blacklist_placement', 'notify_only', 'platform_pause'));

ALTER TABLE margin_guard_policies
    DROP CONSTRAINT IF EXISTS margin_guard_policies_cooldown_sec_check;

ALTER TABLE margin_guard_policies
    ADD CONSTRAINT margin_guard_policies_cooldown_sec_check
    CHECK (cooldown_sec >= 0 AND cooldown_sec <= 604800);

-- +goose Down
ALTER TABLE margin_guard_policies
    DROP CONSTRAINT IF EXISTS margin_guard_policies_cooldown_sec_check;

ALTER TABLE margin_guard_policies
    DROP CONSTRAINT IF EXISTS margin_guard_policies_enforcement_check;

ALTER TABLE margin_guard_policies
    DROP COLUMN IF EXISTS platform_network,
    DROP COLUMN IF EXISTS platform_pause,
    DROP COLUMN IF EXISTS cooldown_sec,
    DROP COLUMN IF EXISTS enforcement;
