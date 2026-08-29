-- +goose Up
CREATE TABLE IF NOT EXISTS team_member_limits (
    user_id UUID PRIMARY KEY,
    customer_id UUID NOT NULL,
    spend_cap_micro BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_team_member_limits_customer ON team_member_limits (customer_id);

CREATE TABLE IF NOT EXISTS team_budget_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL,
    user_id UUID NOT NULL,
    campaign_id UUID NOT NULL REFERENCES campaigns (id) ON DELETE CASCADE,
    requested_budget_micro BIGINT NOT NULL,
    previous_budget_micro BIGINT NOT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    resolved_by UUID
);

CREATE INDEX IF NOT EXISTS idx_team_budget_approvals_pending
    ON team_budget_approvals (customer_id, status)
    WHERE status = 'PENDING';

-- +goose Down
DROP TABLE IF EXISTS team_budget_approvals;
DROP TABLE IF EXISTS team_member_limits;
