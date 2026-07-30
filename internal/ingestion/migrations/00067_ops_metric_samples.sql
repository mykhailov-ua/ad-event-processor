-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS ops;

CREATE TABLE ops.metric_samples (
    name        TEXT NOT NULL,
    labels_hash TEXT NOT NULL DEFAULT '',
    ts          TIMESTAMPTZ NOT NULL,
    value       DOUBLE PRECISION NOT NULL,
    PRIMARY KEY (name, labels_hash, ts)
);

CREATE INDEX idx_ops_metric_samples_name_ts
    ON ops.metric_samples (name, ts DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS ops.metric_samples;
DROP SCHEMA IF EXISTS ops;
-- +goose StatementEnd
