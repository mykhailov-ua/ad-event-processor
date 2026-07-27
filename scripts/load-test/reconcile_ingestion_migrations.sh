#!/usr/bin/env bash
# Repair common local load-test DB drift: migrations marked applied but schema incomplete.
set -euo pipefail

source "$(cd "$(dirname "$0")/../lib" && pwd)/paths.sh"

if [[ -f "$ROOT/.env" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$ROOT/.env"
	set +a
fi

DB_PORT="${DB_PORT:-5430}"
DB_CONTAINER="${DB_CONTAINER:-espx-db-1}"
DB_USER="${DB_USER:-ad_event_processor_user}"
DB_NAME="${DB_NAME:-ad_event_processor}"

log() { printf 'reconcile-ingestion-migrations: %s\n' "$*"; }

psql_exec() {
	docker exec -i "$DB_CONTAINER" psql -h localhost -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" "$@"
}

log "marking 00020 applied when campaigns.budget_limit is already bigint"
psql_exec -v ON_ERROR_STOP=1 <<'SQL'
INSERT INTO public.espx_migrations (filename)
SELECT '00020_change_money_to_bigint.sql'
WHERE NOT EXISTS (
    SELECT 1 FROM public.espx_migrations WHERE filename = '00020_change_money_to_bigint.sql'
)
AND EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'public'
      AND table_name = 'campaigns'
      AND column_name = 'budget_limit'
      AND data_type = 'bigint'
);
SQL

log "repairing events.user_id (00010 drift on partitioned events)"
psql_exec -v ON_ERROR_STOP=1 <<'SQL'
ALTER TABLE events ADD COLUMN IF NOT EXISTS user_id TEXT;
CREATE INDEX IF NOT EXISTS idx_events_user_id ON events(user_id);
SQL

log "repairing campaigns.fraud columns (00026 drift)"
psql_exec -v ON_ERROR_STOP=1 <<'SQL'
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS fraud_threshold_pass SMALLINT NOT NULL DEFAULT 30,
    ADD COLUMN IF NOT EXISTS fraud_threshold_suspect SMALLINT NOT NULL DEFAULT 60,
    ADD COLUMN IF NOT EXISTS fraud_threshold_ivt SMALLINT NOT NULL DEFAULT 80,
    ADD COLUMN IF NOT EXISTS fraud_threshold_block SMALLINT NOT NULL DEFAULT 100,
    ADD COLUMN IF NOT EXISTS ghost_ivt_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS behavior_flags INTEGER NOT NULL DEFAULT 0;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'campaigns_fraud_thresholds_ordered'
    ) THEN
        ALTER TABLE campaigns
            ADD CONSTRAINT campaigns_fraud_thresholds_ordered CHECK (
                fraud_threshold_pass <= fraud_threshold_suspect
                AND fraud_threshold_suspect <= fraud_threshold_ivt
                AND fraud_threshold_ivt <= fraud_threshold_block
            );
    END IF;
END $$;
SQL

log "done"
