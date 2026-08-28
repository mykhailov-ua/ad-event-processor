-- +goose Up
CREATE TABLE IF NOT EXISTS campaign_wizard_sessions (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    owner_user_id UUID,
    current_step TEXT NOT NULL,
    completed_steps JSONB NOT NULL DEFAULT '[]',
    step_payload JSONB NOT NULL DEFAULT '{}',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_campaign_wizard_sessions_customer ON campaign_wizard_sessions(customer_id);
CREATE INDEX IF NOT EXISTS idx_campaign_wizard_sessions_expires ON campaign_wizard_sessions(expires_at);

-- +goose Down
DROP TABLE IF EXISTS campaign_wizard_sessions;
