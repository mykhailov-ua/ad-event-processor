-- +goose Up
-- +goose StatementBegin
ALTER TABLE landers
    ALTER COLUMN url DROP NOT NULL;

CREATE TABLE IF NOT EXISTS lander_assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lander_id UUID NOT NULL REFERENCES landers (id) ON DELETE CASCADE,
    version INT NOT NULL,
    entry_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (lander_id, version)
);

ALTER TABLE landers
    ADD COLUMN IF NOT EXISTS hosted_asset_id UUID REFERENCES lander_assets (id);

CREATE INDEX IF NOT EXISTS lander_assets_lander_id_idx ON lander_assets (lander_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE landers DROP COLUMN IF EXISTS hosted_asset_id;
DROP TABLE IF EXISTS lander_assets;
ALTER TABLE landers ALTER COLUMN url SET NOT NULL;
-- +goose StatementEnd
