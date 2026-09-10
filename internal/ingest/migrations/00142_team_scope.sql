-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS teams (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_teams_customer_id ON teams(customer_id);

ALTER TABLE customers
    ADD COLUMN IF NOT EXISTS team_enforce_ownership BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS postback_inbound_ip_allowlist TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS postback_inbound_secret_encrypted BYTEA;

CREATE INDEX IF NOT EXISTS idx_customers_team_enforce ON customers(team_enforce_ownership) WHERE team_enforce_ownership;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE customers
    DROP COLUMN IF EXISTS postback_inbound_secret_encrypted,
    DROP COLUMN IF EXISTS postback_inbound_ip_allowlist,
    DROP COLUMN IF EXISTS team_enforce_ownership;

DROP TABLE IF EXISTS teams;
-- +goose StatementEnd
