-- +goose Up
-- +goose StatementBegin
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS owner_user_id UUID;

CREATE INDEX IF NOT EXISTS idx_campaigns_owner_user_id ON campaigns(owner_user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_campaigns_owner_user_id;
ALTER TABLE campaigns DROP COLUMN IF EXISTS owner_user_id;
-- +goose StatementEnd
