-- +goose Up
-- +goose StatementBegin
ALTER TABLE campaigns
    RENAME COLUMN l1_cidr_block_enabled TO cidr_block_enabled;
ALTER TABLE campaigns
    RENAME COLUMN l15_proxy_vpn_block_enabled TO proxy_vpn_block_enabled;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE campaigns
    RENAME COLUMN cidr_block_enabled TO l1_cidr_block_enabled;
ALTER TABLE campaigns
    RENAME COLUMN proxy_vpn_block_enabled TO l15_proxy_vpn_block_enabled;
-- +goose StatementEnd
