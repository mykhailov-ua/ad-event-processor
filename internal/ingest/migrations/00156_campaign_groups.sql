-- +goose Up
CREATE TABLE campaign_groups (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id     UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    default_flow_id UUID REFERENCES flows(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_campaign_groups_customer_name_active
    ON campaign_groups (customer_id, lower(name))
    WHERE deleted_at IS NULL;

CREATE INDEX idx_campaign_groups_customer_id ON campaign_groups(customer_id)
    WHERE deleted_at IS NULL;

ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS campaign_group_id UUID REFERENCES campaign_groups(id) ON DELETE SET NULL;

CREATE INDEX idx_campaigns_campaign_group_id ON campaigns(campaign_group_id)
    WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_campaigns_campaign_group_id;

ALTER TABLE campaigns
    DROP COLUMN IF EXISTS campaign_group_id;

DROP INDEX IF EXISTS idx_campaign_groups_customer_id;
DROP INDEX IF EXISTS idx_campaign_groups_customer_name_active;
DROP TABLE IF EXISTS campaign_groups;
