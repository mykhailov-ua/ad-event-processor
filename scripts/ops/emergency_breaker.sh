#!/usr/bin/env bash
# Role: Toggle emergency_breaker via Postgres + UPDATE_SETTINGS outbox (control-plane canonical path).
# Execution context: Operator incident response on control host; sources env file for DB_DSN.
# Env knobs: DB_DSN (required); ENV_FILE (default .env).
# Verify: bash scripts/ops/emergency_breaker.sh off "drill complete" (requires DB + outbox worker)
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

usage() {
  cat <<'EOF'
Usage: emergency_breaker.sh on|off "reason"

Enqueues UPDATE_SETTINGS outbox row and updates system_settings.emergency_breaker.
Requires psql and DB_DSN in env file. Outbox worker must run to propagate to Redis trackers.
EOF
}

if [[ $# -lt 2 ]]; then
  usage >&2
  exit 1
fi

STATE="$1"
REASON="$2"
ENV_FILE="${ENV_FILE:-$ROOT/.env}"

case "$STATE" in
  on|off) ;;
  *)
    usage >&2
    exit 1
    ;;
esac

if [[ -f "$ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  . "$ENV_FILE"
  set +a
fi

if [[ -z "${DB_DSN:-}" ]]; then
  printf 'emergency-breaker: ERROR: DB_DSN unset (source %s)\n' "$ENV_FILE" >&2
  exit 1
fi

if ! command -v psql >/dev/null 2>&1; then
  printf 'emergency-breaker: ERROR: psql not found\n' >&2
  exit 1
fi

VAL="false"
if [[ "$STATE" == "on" ]]; then
  VAL="true"
fi

PAYLOAD=$(printf '{"settings":{"emergency_breaker":"%s"}}' "$VAL")

log() { printf 'emergency-breaker: %s\n' "$*"; }

log "setting emergency_breaker=${VAL} reason=${REASON}"

psql "$DB_DSN" -v ON_ERROR_STOP=1 <<SQL
BEGIN;
INSERT INTO system_settings (key, value)
VALUES ('emergency_breaker', '${VAL}')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;
INSERT INTO outbox_events (event_type, payload, status)
VALUES ('UPDATE_SETTINGS', '${PAYLOAD}'::jsonb, 'PENDING');
COMMIT;
SQL

log "queued UPDATE_SETTINGS outbox event; confirm Redis config:values and /api/v1/ops/shards"
