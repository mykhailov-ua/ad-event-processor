
-- +goose Up
-- +goose StatementBegin
CREATE TABLE campaign_quotas (
    shard_id        SMALLINT NOT NULL,
    campaign_id     UUID NOT NULL,
    reserved_amount BIGINT NOT NULL DEFAULT 0 CHECK (reserved_amount >= 0),
    chunk_size      BIGINT NOT NULL DEFAULT 0 CHECK (chunk_size >= 0),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (shard_id, campaign_id)
);

CREATE INDEX idx_campaign_quotas_campaign_id ON campaign_quotas (campaign_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS campaign_quotas;
-- +goose StatementEnd
