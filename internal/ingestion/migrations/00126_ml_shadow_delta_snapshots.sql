-- +goose Up
CREATE TABLE IF NOT EXISTS ml_shadow_delta_snapshots (
    snapshot_key TEXT PRIMARY KEY DEFAULT 'current',
    range_from TIMESTAMPTZ NOT NULL,
    range_to TIMESTAMPTZ NOT NULL,
    rows JSONB NOT NULL DEFAULT '[]'::jsonb,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE report_jobs DROP CONSTRAINT IF EXISTS report_jobs_status_check;
ALTER TABLE report_jobs ADD CONSTRAINT report_jobs_status_check
    CHECK (status IN ('PENDING', 'RUNNING', 'COMPLETED', 'FAILED', 'CANCELLED'));

-- +goose Down
ALTER TABLE report_jobs DROP CONSTRAINT IF EXISTS report_jobs_status_check;
ALTER TABLE report_jobs ADD CONSTRAINT report_jobs_status_check
    CHECK (status IN ('PENDING', 'RUNNING', 'COMPLETED', 'FAILED'));
DROP TABLE IF EXISTS ml_shadow_delta_snapshots;
