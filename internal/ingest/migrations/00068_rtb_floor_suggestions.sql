-- +goose Up
-- +goose StatementBegin
CREATE TABLE rtb_floor_suggestions (
    placement_id        TEXT PRIMARY KEY,
    deal_id             TEXT NOT NULL,
    current_floor_micro BIGINT NOT NULL,
    suggested_floor_micro BIGINT NOT NULL,
    win_rate            DOUBLE PRECISION NOT NULL DEFAULT 0,
    sample_n            BIGINT NOT NULL DEFAULT 0,
    floor_bucket_micro  BIGINT NOT NULL DEFAULT 0,
    computed_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_rtb_floor_suggestions_computed_at
    ON rtb_floor_suggestions (computed_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS rtb_floor_suggestions;
-- +goose StatementEnd
