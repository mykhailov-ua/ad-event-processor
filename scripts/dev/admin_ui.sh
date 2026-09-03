#!/usr/bin/env bash
# Role: Start, stop, and inspect local admin UI dev stack (compose control + web dev server).
# Execution context: Operator laptop; `up` starts ingest-only compose (db, redis, control) then web on :5173.
# Env knobs: MANAGEMENT_PORT (8188); ADMIN_DEV_PORT (5173); ADMIN_API_PROXY for web -> control.
# Verify: bash scripts/dev/admin_ui.sh up && curl -sf http://127.0.0.1:8188/health
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/go.sh"

VAR_DIR="$ROOT/var"
CONTROL_PID="$VAR_DIR/admin_control.pid"
CONTROL_LOG="$VAR_DIR/admin_control.log"
WEB_PID="$VAR_DIR/admin_web.pid"
WEB_LOG="$VAR_DIR/admin_web.log"
MANAGEMENT_PORT="${MANAGEMENT_PORT:-8188}"
ADMIN_DEV_PORT="${ADMIN_DEV_PORT:-5173}"
CONTROL_URL="${CONTROL_URL:-http://127.0.0.1:${MANAGEMENT_PORT}}"

log() {
  printf 'aed-admin: %s\n' "$*"
}

die() {
  log "ERROR: $*"
  exit 1
}

usage() {
  cat <<EOF
Usage: bash scripts/dev/admin_ui.sh <command>

Commands:
  up        Start ingest-only stack (db, redis, control) + web dev server (:${ADMIN_DEV_PORT})
  down      Stop web dev server (and local control if running); compose stack stays up
  status    Print service URLs, PIDs, and log paths
  stack     Start ingest-only compose only (db, redis, control on :${MANAGEMENT_PORT})
  seed      Bootstrap admin user + UI demo stats for charts
  buyer     Seed buyer dashboard demo (PG stats + ClickHouse economics)
  demo      Re-seed campaign stats and spend for admin charts only
  rebuild-control  Rebuild docker control image (CONTROL_BUILD_TIMEOUT, default 45m) and recreate with control-dev overlay
  control   Run control plane locally in foreground (stops docker control on :${MANAGEMENT_PORT} first)
  web       Run web dev server in foreground (blocks terminal)
  logs      Tail logs (control|web|compose|all; default all)

Aliases (add once per shell):
  source scripts/dev/admin_ui_aliases.sh
  aed-admin up
EOF
}

load_dotenv() {
  if [[ ! -f "$ROOT/.env" ]]; then
    if [[ -f "$ROOT/.env.example" ]]; then
      cp "$ROOT/.env.example" "$ROOT/.env"
      log "created .env from .env.example"
    fi
  fi
  if [[ ! -f "$ROOT/.env" ]]; then
    return 0
  fi
  set -a
  # shellcheck disable=SC1091
  source "$ROOT/.env" 2> /dev/null || log "WARN: .env present but not sourced (parse error)"
  set +a
}

pid_alive() {
  local pid_file="$1"
  local pid=""
  [[ -f "$pid_file" ]] || return 1
  pid="$(<"$pid_file")"
  [[ -n "$pid" ]] || return 1
  kill -0 "$pid" 2> /dev/null
}

control_healthy() {
  curl -sf "${CONTROL_URL}/health" > /dev/null 2>&1
}

web_healthy() {
  curl -sf "http://127.0.0.1:${ADMIN_DEV_PORT}/" > /dev/null 2>&1
}

wait_control_health() {
  local attempt=0
  local max_attempts=90
  while [[ "$attempt" -lt "$max_attempts" ]]; do
    if control_healthy; then
      return 0
    fi
    sleep 2
    attempt=$((attempt + 1))
  done
  return 1
}

ensure_stack() {
  load_dotenv
  if control_healthy; then
    log "control already healthy at ${CONTROL_URL}"
    return 0
  fi
  log "starting ingest-only stack (db, redis, control)..."
  bash "$SCRIPTS/dev/stack/stack.sh" ingest-only
  if ! wait_control_health; then
    log "compose control logs:"
    docker logs ad-event-processor-control-1 2>&1 | tail -20 || true
    die "control did not become healthy at ${CONTROL_URL}"
  fi
  log "control healthy at ${CONTROL_URL}"
}

ensure_control_dev_ui_redirect() {
  local headers
  headers="$(curl -sI "${CONTROL_URL}/login" 2>/dev/null || true)"
  if echo "$headers" | grep -qi "location:.*127.0.0.1:${ADMIN_DEV_PORT}"; then
    return 0
  fi
  compose_control_args
  log "recreating control for ADMIN_UI_DEV_URL redirect..."
  docker compose "${COMPOSE_CONTROL_ARGS[@]}" --profile ingest_only up -d --no-build --force-recreate control
  if ! wait_control_health; then
    die "control did not become healthy after recreate"
  fi
  headers="$(curl -sI "${CONTROL_URL}/login" 2>/dev/null || true)"
  if echo "$headers" | grep -qi "location:.*127.0.0.1:${ADMIN_DEV_PORT}"; then
    return 0
  fi
  log "WARN: :8188 still serves embedded UI; run: bash scripts/dev/admin_ui.sh rebuild-control"
}

ensure_seeded() {
  if ! control_healthy; then
    die "control not healthy; run: bash scripts/dev/admin_ui.sh stack"
  fi
  bash "$SCRIPTS/dev/stack/seed_admin.sh" --no-up
}

ensure_ui_demo() {
  if ! control_healthy; then
    die "control not healthy; run: bash scripts/dev/admin_ui.sh stack"
  fi
  bash "$SCRIPTS/dev/stack/seed_ui_demo.sh"
}

compose_control_args() {
  COMPOSE_CONTROL_ARGS=(--project-directory "$ROOT" -f "$ROOT/docker-compose.yaml" -f "$ROOT/deploy/compose/docker-compose.memory-dev.yaml" -f "$ROOT/deploy/compose/docker-compose.control-dev.yaml")
  if [[ -f "$ROOT/.env" ]]; then
    COMPOSE_CONTROL_ARGS+=(--env-file "$ROOT/.env")
  fi
  if [[ -f "$ROOT/install.compose.env" ]]; then
    COMPOSE_CONTROL_ARGS+=(--env-file "$ROOT/install.compose.env")
  fi
}

rebuild_control() {
  local build_timeout="${CONTROL_BUILD_TIMEOUT:-45m}"
  local health_timeout="${CONTROL_HEALTH_TIMEOUT:-3m}"
  compose_control_args
  export SKIP_CODEGEN="${SKIP_CODEGEN:-1}"
  export DOCKER_BUILDKIT="${DOCKER_BUILDKIT:-1}"
  log "building control (SKIP_CODEGEN=${SKIP_CODEGEN}, timeout ${build_timeout})..."
  if ! timeout "$build_timeout" docker compose "${COMPOSE_CONTROL_ARGS[@]}" --profile ingest_only build control; then
    die "control image build failed or timed out after ${build_timeout}"
  fi
  log "recreating control with control-dev overlay..."
  docker compose "${COMPOSE_CONTROL_ARGS[@]}" --profile ingest_only up -d --no-build --force-recreate control
  log "waiting for control health (timeout ${health_timeout})..."
  if ! timeout "$health_timeout" bash -c "until curl -sf '${CONTROL_URL}/health' >/dev/null; do sleep 3; done"; then
    docker logs ad-event-processor-control-1 2>&1 | tail -20 >&2 || true
    die "control did not become healthy within ${health_timeout}"
  fi
  log "control rebuilt and healthy at ${CONTROL_URL}"
}

stop_docker_control() {
  if ! command -v docker > /dev/null 2>&1; then
    return 0
  fi
  docker compose --project-directory "$ROOT" stop control 2> /dev/null || true
}

start_local_control() {
  if pid_alive "$CONTROL_PID"; then
    log "local control already running (pid $(<"$CONTROL_PID"))"
    return 0
  fi
  load_dotenv
  stop_docker_control
  mkdir -p "$VAR_DIR"
  : >"$CONTROL_LOG"
  (
    cd "$ROOT"
    aed_go_run ./cmd/control
  ) >>"$CONTROL_LOG" 2>&1 &
  echo $! >"$CONTROL_PID"
  if ! wait_control_health; then
    tail -20 "$CONTROL_LOG" >&2 || true
    die "local control failed to start; see $CONTROL_LOG"
  fi
  log "local control running pid=$(<"$CONTROL_PID") -> ${CONTROL_URL}"
}

start_web() {
  if pid_alive "$WEB_PID" && web_healthy; then
    log "web already running (pid $(<"$WEB_PID"))"
    return 0
  fi
  if pid_alive "$WEB_PID"; then
    log "web pid stale or unhealthy; restarting"
    stop_one web "$WEB_PID"
  fi
  if [[ ! -d "$ROOT/web/node_modules" ]]; then
    log "installing web dependencies (npm ci)..."
    (cd "$ROOT/web" && npm ci)
  fi
  load_dotenv
  mkdir -p "$VAR_DIR"
  : >"$WEB_LOG"
  (
    cd "$ROOT/web"
    export ADMIN_API_PROXY="${ADMIN_API_PROXY:-${CONTROL_URL}}"
    export ADMIN_DEV_PORT
    npm run dev
  ) >>"$WEB_LOG" 2>&1 &
  echo $! >"$WEB_PID"
  local attempt=0
  while [[ "$attempt" -lt 60 ]]; do
    if pid_alive "$WEB_PID" && web_healthy; then
      log "web running pid=$(<"$WEB_PID") -> http://127.0.0.1:${ADMIN_DEV_PORT}"
      return 0
    fi
    if ! pid_alive "$WEB_PID"; then
      tail -30 "$WEB_LOG" >&2 || true
      die "web dev server failed to start; see $WEB_LOG"
    fi
    sleep 1
    attempt=$((attempt + 1))
  done
  tail -30 "$WEB_LOG" >&2 || true
  die "web dev server not responding on :${ADMIN_DEV_PORT}; see $WEB_LOG"
}

stop_one() {
  local name="$1"
  local pid_file="$2"
  if ! pid_alive "$pid_file"; then
    rm -f "$pid_file"
    log "$name not running"
    return 0
  fi
  local pid
  pid="$(<"$pid_file")"
  kill "$pid" 2> /dev/null || true
  for _ in 1 2 3 4 5; do
    if ! kill -0 "$pid" 2> /dev/null; then
      break
    fi
    sleep 0.2
  done
  if kill -0 "$pid" 2> /dev/null; then
    kill -9 "$pid" 2> /dev/null || true
  fi
  rm -f "$pid_file"
  log "$name stopped"
}

print_status() {
  if control_healthy; then
    log "control: healthy ${CONTROL_URL}"
  elif pid_alive "$CONTROL_PID"; then
    log "control: local pid=$(<"$CONTROL_PID") (not healthy yet) ${CONTROL_URL}"
  else
    log "control: not reachable ${CONTROL_URL}"
  fi
  if pid_alive "$WEB_PID" && web_healthy; then
    log "web: running pid=$(<"$WEB_PID") http://127.0.0.1:${ADMIN_DEV_PORT}"
  elif pid_alive "$WEB_PID"; then
    log "web: pid=$(<"$WEB_PID") unhealthy on :${ADMIN_DEV_PORT}"
  else
    log "web: stopped"
  fi
  log "logs: $CONTROL_LOG, $WEB_LOG"
}

print_dev_urls() {
  cat <<EOF

Open admin UI:  http://127.0.0.1:${ADMIN_DEV_PORT}
API (no UI):    ${CONTROL_URL}

EOF
}

tail_logs() {
  local target="${1:-all}"
  case "$target" in
    control)
      if [[ -f "$CONTROL_LOG" ]]; then
        tail -n 40 -f "$CONTROL_LOG"
      else
        docker logs -f ad-event-processor-control-1
      fi
      ;;
    web)
      tail -n 40 -f "$WEB_LOG"
      ;;
    compose)
      docker logs -f ad-event-processor-control-1
      ;;
    all)
      tail -n 20 -f "$CONTROL_LOG" "$WEB_LOG"
      ;;
    *)
      die "unknown logs target: $target (use control|web|compose|all)"
      ;;
  esac
}

CMD="${1:-}"
shift || true

case "$CMD" in
  up)
    ensure_stack
    ensure_control_dev_ui_redirect
    ensure_seeded
    ensure_ui_demo
    start_web
    print_status
    print_dev_urls
    ;;
  down)
    stop_one web "$WEB_PID"
    stop_one control "$CONTROL_PID"
    ;;
  status)
    print_status
    ;;
  stack)
    ensure_stack
    ;;
  seed)
    ensure_stack
    ensure_seeded
    ensure_ui_demo
    ;;
  buyer)
    ensure_stack
    ensure_seeded
    bash "$SCRIPTS/dev/stack/seed_buyer_dashboard.sh"
    ;;
  demo)
    ensure_stack
    ensure_ui_demo
    ;;
  rebuild-control)
    rebuild_control
    print_status
    ;;
  control)
    load_dotenv
    stop_docker_control
    cd "$ROOT"
    exec aed_go_run ./cmd/control
    ;;
  web)
    load_dotenv
    cd "$ROOT/web"
    export ADMIN_API_PROXY="${ADMIN_API_PROXY:-${CONTROL_URL}}"
    export ADMIN_DEV_PORT
    exec npm run dev
    ;;
  logs)
    tail_logs "${1:-all}"
    ;;
  -h | --help | help)
    usage
    exit 0
    ;;
  '')
    usage
    exit 1
    ;;
  *)
    die "unknown command: $CMD (try --help)"
    ;;
esac
