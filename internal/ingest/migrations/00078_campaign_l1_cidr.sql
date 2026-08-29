-- +goose Up
-- +goose StatementBegin
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS l1_cidr_block_enabled BOOLEAN NOT NULL DEFAULT true;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE campaigns
    DROP COLUMN IF EXISTS l1_cidr_block_enabled;
-- +goose StatementEnd
