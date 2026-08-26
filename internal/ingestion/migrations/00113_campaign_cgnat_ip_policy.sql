-- +goose Up
-- +goose StatementBegin
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS cgnat_ip_policy_enabled BOOLEAN NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE campaigns DROP COLUMN IF EXISTS cgnat_ip_policy_enabled;
-- +goose StatementEnd
