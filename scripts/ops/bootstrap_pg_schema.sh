#!/usr/bin/env bash
# Role: Apply tracked Postgres schema migrations before control/tracker/processor start.
# Execution context: stack.sh, ad-event-processor-install.sh, or bare-metal with DB_DSN in .env.
# Env knobs: DB_DSN (required); AD_EVENT_PROCESSOR_REPO_ROOT; AD_EVENT_PROCESSOR_SKIP_PG_MIGRATE=1 to skip.
# Verify: DB_DSN=postgres://... bash scripts/ops/bootstrap_pg_schema.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/installer_env.sh"
source "$SCRIPTS/lib/install_cli.sh"
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

export AD_EVENT_PROCESSOR_REPO_ROOT="$ROOT"

compose_db_running() {
  command -v docker > /dev/null 2>&1 \
    && [[ -n "$(bootstrap_compose ps -q db 2> /dev/null | head -1 || true)" ]]
}

bootstrap_db_port() {
  if compose_db_running; then
    echo "5430"
    return 0
  fi
  local port
  port="$(installer_read_env DB_PORT)"
  echo "${port:-${DB_PORT:-5430}}"
}

bootstrap_db_credentials() {
  local user db_name pass
  user="$(installer_read_env DB_USER)"
  user="${user:-${DB_USER:-ad_event_processor_user}}"
  db_name="$(installer_read_env DB_NAME)"
  db_name="${db_name:-${DB_NAME:-ad_event_processor}}"
  pass="$(installer_read_env DB_PASSWORD)"
  pass="${pass:-${DB_PASSWORD:-}}"
  printf '%s\n%s\n%s\n' "$user" "$db_name" "$pass"
}

bootstrap_resolve_dsn() {
  local user db_name pass port
  port="$(bootstrap_db_port)"
  mapfile -t _bootstrap_db_cred < <(bootstrap_db_credentials)
  user="${_bootstrap_db_cred[0]}"
  db_name="${_bootstrap_db_cred[1]}"
  pass="${_bootstrap_db_cred[2]}"
  printf 'postgres://%s:%s@127.0.0.1:%s/%s?sslmode=disable' "$user" "$pass" "$port" "$db_name"
}

persist_bootstrap_dsn() {
  local dsn port
  dsn="$(bootstrap_resolve_dsn)"
  port="$(bootstrap_db_port)"
  export DB_DSN="$dsn"
  export PAYMENT_DB_DSN="$dsn"
  export DB_PORT="$port"
  if [[ -f "$ROOT/.env" ]]; then
    install_cli_set_env_key "$ROOT/.env" DB_PORT "$port"
    install_cli_set_env_key "$ROOT/.env" DB_DSN "$dsn"
    install_cli_set_env_key "$ROOT/.env" PAYMENT_DB_DSN "$dsn"
  fi
}

migrations_present() {
  [[ -d "$ROOT/internal/ingest/migrations" ]]
}

bootstrap_compose() {
  local -a args=(--project-directory "$ROOT" -f "$ROOT/docker-compose.yaml")
  if [[ -f "$ROOT/.env" ]]; then
    args+=(--env-file "$ROOT/.env")
  fi
  docker compose "${args[@]}" "$@"
}

postgres_client_image() {
  local ver
  ver="$(installer_read_env POSTGRES_VERSION)"
  ver="${ver:-${POSTGRES_VERSION:-16-alpine}}"
  echo "postgres:${ver}"
}

postgres_ready_via_compose_exec() {
  local port="$1" user="$2" db_name="$3"
  bootstrap_compose exec -T db \
    pg_isready -h /run/ad-event-processor/postgresql -p "$port" -U "$user" -d "$db_name" > /dev/null 2>&1
}

postgres_ready_via_tcp() {
  local port="$1" user="$2" db_name="$3"
  if command -v pg_isready > /dev/null 2>&1; then
    pg_isready -h 127.0.0.1 -p "$port" -U "$user" -d "$db_name" > /dev/null 2>&1
    return $?
  fi
  if ! command -v docker > /dev/null 2>&1; then
    return 1
  fi
  docker run --rm --network host "$(postgres_client_image)" \
    pg_isready -h 127.0.0.1 -p "$port" -U "$user" -d "$db_name" > /dev/null 2>&1
}

ensure_compose_postgres_database() {
  local port user db_name pass
  port="$(bootstrap_db_port)"
  mapfile -t _bootstrap_db_cred < <(bootstrap_db_credentials)
  user="${_bootstrap_db_cred[0]}"
  db_name="${_bootstrap_db_cred[1]}"
  pass="${_bootstrap_db_cred[2]}"

  if ! command -v docker > /dev/null 2>&1; then
    return 0
  fi
  if [[ -z "$(bootstrap_compose ps -q db 2> /dev/null | head -1 || true)" ]]; then
    return 0
  fi

  if bootstrap_compose exec -T db \
    psql -h /run/ad-event-processor/postgresql -p "$port" -U "$user" -d "$db_name" -v ON_ERROR_STOP=1 -c 'SELECT 1' > /dev/null 2>&1; then
    return 0
  fi

  echo "bootstrap_pg_schema: creating database ${db_name} in compose db..."
  if bootstrap_compose exec -T db \
    psql -h /run/ad-event-processor/postgresql -p "$port" -U "$user" -d postgres -v ON_ERROR_STOP=1 \
    -c "CREATE DATABASE \"${db_name}\"" > /dev/null 2>&1; then
    return 0
  fi

  if [[ -n "$pass" ]]; then
    PGPASSWORD="$pass" bootstrap_compose exec -T -e PGPASSWORD="$pass" db \
      psql -h /run/ad-event-processor/postgresql -p "$port" -U "$user" -d postgres -v ON_ERROR_STOP=1 \
      -c "CREATE DATABASE \"${db_name}\"" > /dev/null 2>&1 \
      && return 0
  fi

  echo "bootstrap_pg_schema: could not ensure database ${db_name} exists" >&2
  return 1
}

normalize_systemd_db_dsn() {
  persist_bootstrap_dsn
}

wait_for_postgres() {
  local port user db_name host attempts i cid
  port="$(bootstrap_db_port)"
  mapfile -t _bootstrap_db_cred < <(bootstrap_db_credentials)
  user="${_bootstrap_db_cred[0]}"
  db_name="${_bootstrap_db_cred[1]}"
  host="127.0.0.1"
  attempts="${BOOTSTRAP_PG_WAIT_ATTEMPTS:-90}"

  if command -v docker > /dev/null 2>&1; then
    cid="$(bootstrap_compose ps -q db 2> /dev/null | head -1 || true)"
    if [[ -n "$cid" ]]; then
      echo "bootstrap_pg_schema: waiting for compose db (pg_isready)..."
      i=0
      while [[ $i -lt $attempts ]]; do
        if postgres_ready_via_compose_exec "$port" "$user" "$db_name" \
          || postgres_ready_via_tcp "$port" "$user" "$db_name"; then
          echo "bootstrap_pg_schema: postgres ready (compose db)"
          return 0
        fi
        sleep 2
        i=$((i + 1))
      done
      echo "bootstrap_pg_schema: compose db not ready after ${attempts} attempts" >&2
      return 1
    fi
  fi

  echo "bootstrap_pg_schema: waiting for postgres on ${host}:${port}..."
  i=0
  while [[ $i -lt $attempts ]]; do
    if postgres_ready_via_tcp "$port" "$user" "$db_name"; then
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
  local dsn port db_name
  if [[ -n "$ONLY" ]]; then
    args+=(-only "$ONLY")
  fi

  persist_bootstrap_dsn
  dsn="$DB_DSN"
  port="$(bootstrap_db_port)"
  mapfile -t _bootstrap_db_cred < <(bootstrap_db_credentials)
  db_name="${_bootstrap_db_cred[1]}"

  echo "bootstrap_pg_schema: migrate target 127.0.0.1:${port}/${db_name}"

  if [[ -x "$ROOT/bin/migrate-cold-path" ]]; then
    echo "bootstrap_pg_schema: bin/migrate-cold-path"
    env DB_DSN="$dsn" PAYMENT_DB_DSN="$dsn" DB_PORT="$port" \
      "$ROOT/bin/migrate-cold-path" "${args[@]}"
    return 0
  fi

  if migrations_present && aed_go_bin > /dev/null 2>&1; then
    echo "bootstrap_pg_schema: go run ./cmd/migrate-cold-path"
    DB_DSN="$dsn" PAYMENT_DB_DSN="$dsn" DB_PORT="$port" \
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

normalize_systemd_db_dsn
ensure_compose_postgres_database

echo "bootstrap_pg_schema: applying cold-path schema migrations..."
run_migrate
echo "bootstrap_pg_schema: done"
