CREATE TABLE IF NOT EXISTS report_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id),
    report_key TEXT NOT NULL,
    spec JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING',
    file_path TEXT,
    error_message TEXT,
    bytes BIGINT NOT NULL DEFAULT 0,
    idempotency_key TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT report_jobs_status_check CHECK (status IN ('PENDING', 'RUNNING', 'COMPLETED', 'FAILED'))
);

CREATE UNIQUE INDEX IF NOT EXISTS report_jobs_idempotency_key_uidx
    ON report_jobs (idempotency_key)
    WHERE idempotency_key IS NOT NULL AND idempotency_key <> '';

CREATE INDEX IF NOT EXISTS report_jobs_status_created_idx
    ON report_jobs (status, created_at)
    WHERE status IN ('PENDING', 'RUNNING');

CREATE INDEX IF NOT EXISTS report_jobs_customer_created_idx
    ON report_jobs (customer_id, created_at DESC);
