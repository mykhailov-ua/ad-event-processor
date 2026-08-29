-- +goose Up
-- +goose StatementBegin
CREATE TABLE region_traffic_dial (
    region_code   SMALLINT PRIMARY KEY,
    score         DOUBLE PRECISION NOT NULL,
    weight        DOUBLE PRECISION NOT NULL,
    provenance    TEXT NOT NULL,
    epoch_id      BIGINT NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS region_traffic_dial;
-- +goose StatementEnd
