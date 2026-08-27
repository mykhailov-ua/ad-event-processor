#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/load_test_env.sh"

if [[ -f "$ROOT/.env" ]]; then
  set -a
  source "$ROOT/.env"
  set +a
fi

load_test_source_env "$ROOT" 2> /dev/null || true
load_test_export_derived 2> /dev/null || true
load_test_compose COMPOSE "$ROOT"

DB_PORT="${DB_PORT:-5430}"
DB_USER="${DB_USER:-ad_event_processor_user}"
DB_NAME="${DB_NAME:-ad_event_processor}"

log() { printf 'reconcile-ingestion-migrations: %s\n' "$*"; }

psql_exec() {
  "${COMPOSE[@]}" exec -T db psql -h localhost -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" "$@"
}

log "marking 00020 applied when campaigns.budget_limit is already bigint"
psql_exec -v ON_ERROR_STOP=1 << 'SQL'
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_name = 'tracked_migrations'
    ) THEN
        RETURN;
    END IF;
    INSERT INTO public.tracked_migrations (filename)
    SELECT '00020_change_money_to_bigint.sql'
    WHERE NOT EXISTS (
        SELECT 1 FROM public.tracked_migrations WHERE filename = '00020_change_money_to_bigint.sql'
    )
    AND EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'campaigns'
          AND column_name = 'budget_limit'
          AND data_type = 'bigint'
    );
END $$;
SQL

log "repairing events.user_id (00010 drift on partitioned events)"
psql_exec -v ON_ERROR_STOP=1 << 'SQL'
ALTER TABLE events ADD COLUMN IF NOT EXISTS user_id TEXT;
CREATE INDEX IF NOT EXISTS idx_events_user_id ON events(user_id);
SQL

log "repairing campaigns.fraud columns (00026 drift) and silent_reject rename (00101)"
psql_exec -v ON_ERROR_STOP=1 << 'SQL'
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS fraud_threshold_pass SMALLINT NOT NULL DEFAULT 30,
    ADD COLUMN IF NOT EXISTS fraud_threshold_suspect SMALLINT NOT NULL DEFAULT 60,
    ADD COLUMN IF NOT EXISTS fraud_threshold_ivt SMALLINT NOT NULL DEFAULT 80,
    ADD COLUMN IF NOT EXISTS fraud_threshold_block SMALLINT NOT NULL DEFAULT 100,
    ADD COLUMN IF NOT EXISTS behavior_flags INTEGER NOT NULL DEFAULT 0;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'campaigns' AND column_name = 'ghost_ivt_enabled'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'campaigns' AND column_name = 'silent_reject_enabled'
    ) THEN
        ALTER TABLE campaigns RENAME COLUMN ghost_ivt_enabled TO silent_reject_enabled;
    ELSIF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'campaigns' AND column_name = 'silent_reject_enabled'
    ) THEN
        ALTER TABLE campaigns
            ADD COLUMN silent_reject_enabled BOOLEAN NOT NULL DEFAULT FALSE;
    END IF;

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

log "aliasing renumbered ingestion migrations (duplicate version cleanup)"
psql_exec -v ON_ERROR_STOP=1 << 'SQL'
DO $$
DECLARE
    pair TEXT[][];
BEGIN
    pair := ARRAY[
        ARRAY['00042_ml_manual_labels.sql', '00090_ml_manual_labels_customer.sql'],
        ARRAY['00073_telegram_mini_app_url.sql', '00117_telegram_mini_app_url.sql'],
        ARRAY['00090_recon_disc_customer_created_idx.sql', '00118_recon_disc_customer_created_idx.sql'],
        ARRAY['00099_report_views_schedules.sql', '00119_report_views_schedules.sql'],
        ARRAY['00106_platform_campaign_api.sql', '00120_platform_campaign_api.sql']
    ];
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_name = 'tracked_migrations'
    ) THEN
        RETURN;
    END IF;
    FOR i IN 1 .. array_length(pair, 1) LOOP
        INSERT INTO public.tracked_migrations (filename)
        SELECT pair[i][2]
        WHERE EXISTS (
            SELECT 1 FROM public.tracked_migrations WHERE filename = pair[i][1]
        )
        AND NOT EXISTS (
            SELECT 1 FROM public.tracked_migrations WHERE filename = pair[i][2]
        );
        DELETE FROM public.tracked_migrations
        WHERE filename = pair[i][1]
          AND EXISTS (
              SELECT 1 FROM public.tracked_migrations WHERE filename = pair[i][2]
          );
    END LOOP;
END $$;
SQL

log "done"
