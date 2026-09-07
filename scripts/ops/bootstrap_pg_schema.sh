#!/usr/bin/env bash
# Role: Apply tracked Postgres schema migrations before control/tracker/processor start.
# Execution context: stack.sh, ad-event-processor-install.sh, or bare-metal with DB_DSN in .env.
# Env knobs: DB_DSN (required); AD_EVENT_PROCESSOR_REPO_ROOT; AD_EVENT_PROCESSOR_SKIP_PG_MIGRATE=1 to skip.
# Verify: DB_DSN=postgres://... bash scripts/ops/bootstrap_pg_schema.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/installer_env.sh"
# shellcheck source=scripts/lib/go.sh
source "$SCRIPTS/lib/go.sh"
cd "$ROOT"

NO_WAIT=0
ONLY=""

usage() {
  echo "usage: $0 [--no-wait] [--only ads,auth,billing,notifier]" >&2
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --no-wait)
      NO_WAIT=1
      shift
      ;;
    --only)
      ONLY="${2:-}"
      shift 2
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      echo "bootstrap_pg_schema: unknown argument: $1" >&2
      usage
      exit 2
      ;;
  esac
done

if [[ "${AD_EVENT_PROCESSOR_SKIP_PG_MIGRATE:-}" == "1" ]]; then
  echo "bootstrap_pg_schema: skipped (AD_EVENT_PROCESSOR_SKIP_PG_MIGRATE=1)"
  exit 0
fi

if [[ -f "$ROOT/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$ROOT/.env"
  set +a
fi

if [[ -z "${DB_DSN:-}" ]]; then
  echo "bootstrap_pg_schema: DB_DSN is required (.env or environment)" >&2
  exit 1
fi

export AD_EVENT_PROCESSOR_REPO_ROOT="$ROOT"

migrations_present() {
  [[ -d "$ROOT/internal/ingest/migrations" ]]
}

wait_for_postgres() {
  local port user db_name host attempts i
  port="$(installer_read_env DB_PORT)"
  port="${port:-${DB_PORT:-5430}}"
  user="$(installer_read_env DB_USER)"
  user="${user:-${DB_USER:-ad_event_processor_user}}"
  db_name="$(installer_read_env DB_NAME)"
  db_name="${db_name:-${DB_NAME:-ad_event_processor}}"
  host="127.0.0.1"
  attempts="${BOOTSTRAP_PG_WAIT_ATTEMPTS:-90}"

  if command -v docker > /dev/null 2>&1 && docker compose --project-directory "$ROOT" ps db 2> /dev/null | grep -q 'running'; then
    echo "bootstrap_pg_schema: waiting for compose db (pg_isready)..."
    i=0
    while [[ $i -lt $attempts ]]; do
      if docker compose --project-directory "$ROOT" exec -T db \
        pg_isready -h /run/ad-event-processor/postgresql -p "$port" -U "$user" -d "$db_name" > /dev/null 2>&1; then
        echo "bootstrap_pg_schema: postgres ready (compose db)"
        return 0
      fi
      sleep 2
      i=$((i + 1))
    done
    echo "bootstrap_pg_schema: compose db not ready after ${attempts} attempts" >&2
    return 1
  fi

  if ! command -v pg_isready > /dev/null 2>&1; then
    echo "bootstrap_pg_schema: pg_isready not found; skipping wait (set BOOTSTRAP_PG_WAIT_ATTEMPTS=0 to silence)" >&2
    return 0
  fi

  echo "bootstrap_pg_schema: waiting for postgres on ${host}:${port}..."
  i=0
  while [[ $i -lt $attempts ]]; do
    if pg_isready -h "$host" -p "$port" -U "$user" -d "$db_name" > /dev/null 2>&1; then
      echo "bootstrap_pg_schema: postgres ready (${host}:${port})"
      return 0
    fi
    sleep 2
    i=$((i + 1))
  done
  echo "bootstrap_pg_schema: postgres not ready on ${host}:${port}" >&2
  return 1
}

run_migrate() {
  local -a args=()
  if [[ -n "$ONLY" ]]; then
    args+=(-only "$ONLY")
  fi

  if [[ -x "$ROOT/bin/migrate-cold-path" ]]; then
    echo "bootstrap_pg_schema: bin/migrate-cold-path"
    "$ROOT/bin/migrate-cold-path" "${args[@]}"
    return 0
  fi

  if migrations_present && aed_go_bin > /dev/null 2>&1; then
    echo "bootstrap_pg_schema: go run ./cmd/migrate-cold-path"
    aed_go_run ./cmd/migrate-cold-path/ "${args[@]}"
    return 0
  fi

  echo "bootstrap_pg_schema: need bin/migrate-cold-path or Go toolchain with internal/*/migrations at $ROOT" >&2
  exit 1
}

if ! migrations_present; then
  echo "bootstrap_pg_schema: missing internal/ingest/migrations under $ROOT" >&2
  exit 1
fi

if [[ "$NO_WAIT" != "1" ]]; then
  wait_for_postgres
fi

echo "bootstrap_pg_schema: applying cold-path schema migrations..."
run_migrate
echo "bootstrap_pg_schema: done"
