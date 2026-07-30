#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"

if [[ -f "$ROOT/.env" ]]; then
	set -a
	source "$ROOT/.env"
	set +a
fi

DB_PORT="${DB_PORT:-5430}"
DB_CONTAINER="${DB_CONTAINER:-espx-db-1}"
DB_USER="${DB_USER:-ad_event_processor_user}"
DB_NAME="${DB_NAME:-ad_event_processor}"

log() { printf 'verify-load-test-schema: %s\n' "$*"; }
die() { printf 'verify-load-test-schema: ERROR: %s\n' "$*" >&2; exit 1; }

missing="$(
	docker exec -i "$DB_CONTAINER" psql -h localhost -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -At -v ON_ERROR_STOP=1 <<'SQL'
SELECT string_agg(req.column_name, ', ')
FROM (
    VALUES
        ('events', 'user_id'),
        ('campaigns', 'fraud_threshold_pass'),
        ('campaigns', 'budget_limit')
) AS req(table_name, column_name)
WHERE NOT EXISTS (
    SELECT 1
    FROM information_schema.columns c
    WHERE c.table_schema = 'public'
      AND c.table_name = req.table_name
      AND c.column_name = req.column_name
);
SQL
)"

if [[ -n "$missing" ]]; then
	die "missing columns: $missing"
fi

log "schema OK (events.user_id, campaigns.fraud_threshold_pass, campaigns.budget_limit)"
