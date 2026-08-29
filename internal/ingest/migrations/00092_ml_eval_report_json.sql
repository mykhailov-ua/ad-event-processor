-- +goose Up
-- +goose StatementBegin
ALTER TABLE ml_eval_reports
    ADD COLUMN IF NOT EXISTS report_json JSONB NOT NULL DEFAULT '{}';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE ml_eval_reports DROP COLUMN IF EXISTS report_json;
-- +goose StatementEnd
