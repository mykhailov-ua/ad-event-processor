-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS report_saved_views (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id TEXT NOT NULL DEFAULT 'system',
    customer_id UUID NOT NULL REFERENCES customers(id),
    name TEXT NOT NULL,
    report_key TEXT NOT NULL,
    spec JSONB NOT NULL DEFAULT '{}',
    is_shared BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS report_saved_views_customer_idx
    ON report_saved_views (customer_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS report_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id),
    report_key TEXT NOT NULL,
    format TEXT NOT NULL DEFAULT 'csv',
    cron_expr TEXT NOT NULL,
    spec JSONB NOT NULL DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    next_run_at TIMESTAMPTZ NOT NULL,
    last_run_at TIMESTAMPTZ,
    last_job_id UUID REFERENCES report_jobs(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT report_schedules_format_check CHECK (format IN ('csv'))
);

CREATE INDEX IF NOT EXISTS report_schedules_next_run_idx
    ON report_schedules (next_run_at)
    WHERE enabled = TRUE;

CREATE INDEX IF NOT EXISTS report_schedules_customer_idx
    ON report_schedules (customer_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS report_schedules;
DROP TABLE IF EXISTS report_saved_views;
-- +goose StatementEnd
