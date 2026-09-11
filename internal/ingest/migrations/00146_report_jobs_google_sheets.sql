-- +goose Up
-- +goose StatementBegin
ALTER TABLE report_jobs
    ADD COLUMN IF NOT EXISTS spreadsheet_id TEXT,
    ADD COLUMN IF NOT EXISTS spreadsheet_url TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE report_jobs
    DROP COLUMN IF EXISTS spreadsheet_url,
    DROP COLUMN IF EXISTS spreadsheet_id;
-- +goose StatementEnd
