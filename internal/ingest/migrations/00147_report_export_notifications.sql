-- In-app export job notifications (dedup per job_id + user_id).
CREATE TABLE IF NOT EXISTS report_export_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES report_jobs(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    customer_id UUID NOT NULL,
    report_key TEXT,
    kind TEXT NOT NULL CHECK (kind IN ('completed', 'failed')),
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (job_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_report_export_notifications_user_created
    ON report_export_notifications (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_report_export_notifications_user_unread
    ON report_export_notifications (user_id, created_at DESC)
    WHERE read_at IS NULL;
