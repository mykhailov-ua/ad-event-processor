-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'campaigns'
          AND column_name = 'l1_cidr_block_enabled'
    ) THEN
        ALTER TABLE campaigns
            RENAME COLUMN l1_cidr_block_enabled TO cidr_block_enabled;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'campaigns'
          AND column_name = 'l15_proxy_vpn_block_enabled'
    ) THEN
        ALTER TABLE campaigns
            RENAME COLUMN l15_proxy_vpn_block_enabled TO proxy_vpn_block_enabled;
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'campaigns'
          AND column_name = 'cidr_block_enabled'
    ) THEN
        ALTER TABLE campaigns
            RENAME COLUMN cidr_block_enabled TO l1_cidr_block_enabled;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'campaigns'
          AND column_name = 'proxy_vpn_block_enabled'
    ) THEN
        ALTER TABLE campaigns
            RENAME COLUMN proxy_vpn_block_enabled TO l15_proxy_vpn_block_enabled;
    END IF;
END $$;
-- +goose StatementEnd
