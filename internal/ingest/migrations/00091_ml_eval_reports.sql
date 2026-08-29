-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS ml_eval_reports (
    id TEXT PRIMARY KEY,
    generated_at TIMESTAMPTZ NOT NULL,
    precision DOUBLE PRECISION NOT NULL DEFAULT 0,
    recall DOUBLE PRECISION NOT NULL DEFAULT 0,
    drift_json JSONB NOT NULL DEFAULT '{}',
    status TEXT NOT NULL CHECK (status IN ('ok', 'empty', 'error', 'skipped')),
    label_method TEXT NOT NULL DEFAULT 'proxy',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS ml_eval_reports;
-- +goose StatementEnd
