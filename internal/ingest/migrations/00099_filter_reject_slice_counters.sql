-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS ops.filter_reject_slice_counters (
    slice_key     TEXT PRIMARY KEY,
    counter_value DOUBLE PRECISION NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS ops.filter_reject_slice_counters;
-- +goose StatementEnd
