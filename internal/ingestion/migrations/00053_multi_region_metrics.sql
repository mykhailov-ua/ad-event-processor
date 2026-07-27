-- +goose Up
-- +goose StatementBegin
CREATE TABLE node_metric_buckets (
    node_id       TEXT NOT NULL,
    region_code   SMALLINT NOT NULL,
    role          TEXT NOT NULL,
    bucket_ts     TIMESTAMPTZ NOT NULL,
    metric        TEXT NOT NULL,
    value_p50     DOUBLE PRECISION,
    value_p99     DOUBLE PRECISION,
    value_mean    DOUBLE PRECISION,
    sample_count  BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (node_id, bucket_ts, metric)
);

CREATE INDEX idx_node_metric_buckets_region_role_ts
    ON node_metric_buckets (region_code, role, bucket_ts DESC);

CREATE TABLE node_metric_daily_snapshots (
    day           DATE NOT NULL,
    region_code   SMALLINT NOT NULL,
    role          TEXT NOT NULL,
    metric        TEXT NOT NULL,
    value_p50     DOUBLE PRECISION,
    value_p99     DOUBLE PRECISION,
    value_mean    DOUBLE PRECISION,
    sample_count  BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (day, region_code, role, metric)
);

CREATE TABLE node_capacity_scores (
    node_id       TEXT NOT NULL,
    region_code   SMALLINT NOT NULL,
    role          TEXT NOT NULL,
    score         DOUBLE PRECISION NOT NULL,
    weight        DOUBLE PRECISION NOT NULL,
    provenance    TEXT NOT NULL,
    epoch_id      BIGINT NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (node_id, region_code, role)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS node_capacity_scores;
DROP TABLE IF EXISTS node_metric_daily_snapshots;
DROP TABLE IF EXISTS node_metric_buckets;
-- +goose StatementEnd
