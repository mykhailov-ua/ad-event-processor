#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ -f "$ROOT/.env" ]]; then
  set -a

  source "$ROOT/.env"
  set +a
fi

RUN_DIR="${UDS_SMOKE_RUN_DIR:-/run/ad-event-processor}"
TRACKER_COUNT="${UDS_SMOKE_TRACKER_COUNT:-4}"
COMPOSE_FILE="${UDS_SMOKE_COMPOSE_FILE:-deploy/compose/docker-compose.yaml}"
TRANSPORT_USE_UDS="${TRANSPORT_USE_UDS:-1}"
REDIS_PASSWORD="${REDIS_PASSWORD:-}"
DB_PORT="${DB_PORT:-5430}"
DB_USER="${DB_USER:-ad_event_processor_user}"
DB_NAME="${DB_NAME:-ad_event_processor}"
CH_USER="${CH_USER:-default}"
CH_PASSWORD="${CH_PASSWORD:-}"
CH_NAME="${CH_NAME:-ad_event_processor}"
CH_IMAGE="${CH_VERSION:+clickhouse/clickhouse-server:${CH_VERSION}}"
CH_IMAGE="${CH_IMAGE:-clickhouse/clickhouse-server:24.8}"

log() { printf 'uds-transport-smoke: %s\n' "$*"; }
die() {
  printf 'uds-transport-smoke: ERROR: %s\n' "$*" >&2
  exit 1
}

if [[ "$TRANSPORT_USE_UDS" == "0" && "${UDS_SMOKE_FORCE:-0}" != "1" ]]; then
  log "skip (TRANSPORT_USE_UDS=0)"
  exit 0
fi

if [[ "${UDS_SMOKE_OFFLINE:-0}" != "1" ]]; then
  if ! command -v docker > /dev/null 2>&1 || ! docker info > /dev/null 2>&1; then
    log "skip (docker unavailable for live unix probes)"
    exit 0
  fi
fi

COMPOSE=(docker compose -f "$COMPOSE_FILE")
RUN_VOL=""
TRACKER_CID=""

socket_exists() {
  local path="$1"
  if [[ -S "$path" ]]; then
    return 0
  fi
  if [[ -n "$RUN_VOL" ]]; then
    docker run --rm -v "${RUN_VOL}:/run/ad-event-processor:ro" alpine:3.20 test -S "$path"
    return
  fi
  return 1
}

resolve_run_volume() {
  local cid
  cid="$("${COMPOSE[@]}" ps -q tracker-0 2> /dev/null | head -1 || true)"
  if [[ -z "$cid" ]]; then
    return 1
  fi
  TRACKER_CID="$cid"
  RUN_VOL="$(docker inspect "$cid" --format '{{range .Mounts}}{{if eq .Destination "/run/ad-event-processor"}}{{.Name}}{{end}}{{end}}')"
  [[ -n "$RUN_VOL" ]]
}

http_unix_probe() {
  local sock="$1"
  local path="${2:-/health}"
  if [[ "${UDS_SMOKE_OFFLINE:-0}" == "1" ]]; then
    [[ -S "$sock" ]]
    return
  fi
  if [[ -n "$RUN_VOL" ]]; then
    docker run --rm -v "${RUN_VOL}:/run/ad-event-processor:ro" curlimages/curl:8.5.0 \
      -sf --max-time 5 --unix-socket "$sock" "http://localhost${path}" > /dev/null
    return
  fi
  if command -v curl > /dev/null 2>&1 && [[ -S "$sock" ]]; then
    curl -sf --max-time 5 --unix-socket "$sock" "http://localhost${path}" > /dev/null
    return
  fi
  return 1
}

redis_unix_ping() {
  local sock="$1"
  if [[ "${UDS_SMOKE_OFFLINE:-0}" == "1" ]]; then
    [[ -S "$sock" ]]
    return
  fi
  if [[ -n "$RUN_VOL" ]]; then
    docker run --rm -v "${RUN_VOL}:/run/ad-event-processor:ro" redis:7-alpine \
      redis-cli -s "$sock" -a "$REDIS_PASSWORD" --no-auth-warning PING 2> /dev/null | grep -q PONG
    return
  fi
  if command -v redis-cli > /dev/null 2>&1 && [[ -S "$sock" ]]; then
    redis-cli -s "$sock" -a "$REDIS_PASSWORD" --no-auth-warning PING 2> /dev/null | grep -q PONG
    return
  fi
  return 1
}

pg_unix_ready() {
  local sock_dir="$1"
  local port="$2"
  if [[ "${UDS_SMOKE_OFFLINE:-0}" == "1" ]]; then
    [[ -d "$sock_dir" ]]
    return
  fi
  if [[ -n "$RUN_VOL" ]]; then
    docker run --rm -v "${RUN_VOL}:/run/ad-event-processor:ro" postgres:16-alpine \
      pg_isready -h "$sock_dir" -p "$port" -U "$DB_USER" -d "$DB_NAME" > /dev/null 2>&1
    return
  fi
  if command -v pg_isready > /dev/null 2>&1 && [[ -d "$sock_dir" ]]; then
    pg_isready -h "$sock_dir" -p "$port" -U "$DB_USER" -d "$DB_NAME" > /dev/null 2>&1
    return
  fi
  return 1
}

ch_unix_query() {
  local sock="$1"
  if [[ "${UDS_SMOKE_OFFLINE:-0}" == "1" ]]; then
    [[ -S "$sock" ]]
    return
  fi
  if [[ -n "$RUN_VOL" ]]; then
    docker run --rm -v "${RUN_VOL}:/run/ad-event-processor:ro" "$CH_IMAGE" \
      clickhouse-client --host "$sock" --user "$CH_USER" --password "$CH_PASSWORD" \
      --query "SELECT 1" > /dev/null 2>&1
    return
  fi
  if command -v clickhouse-client > /dev/null 2>&1 && [[ -S "$sock" ]]; then
    clickhouse-client --host "$sock" --user "$CH_USER" --password "$CH_PASSWORD" \
      --query "SELECT 1" > /dev/null 2>&1
    return
  fi
  return 1
}

tracker_health_probe() {
  local sock="$1"
  if [[ "${UDS_SMOKE_OFFLINE:-0}" == "1" ]]; then
    [[ -S "$sock" ]]
    return
  fi
  if [[ -n "$TRACKER_CID" ]]; then
    docker exec "$TRACKER_CID" /tracker --health-probe "$sock" > /dev/null 2>&1
    return
  fi
  if [[ -x "$ROOT/bin/tracker" ]] && [[ -S "$sock" ]]; then
    "$ROOT/bin/tracker" --health-probe "$sock" > /dev/null 2>&1
    return
  fi
  http_unix_probe "$sock" "/health"
}

check() {
  local name="$1"
  shift
  if "$@"; then
    log "ok  $name"
    PASS=$((PASS + 1))
  else
    log "FAIL $name"
    FAIL=$((FAIL + 1))
  fi
}

PASS=0
FAIL=0

if resolve_run_volume; then
  log "compose volume=$RUN_VOL (tracker-0=$TRACKER_CID)"
elif [[ "${UDS_SMOKE_OFFLINE:-0}" == "1" ]]; then
  log "offline mode; probing host paths under $RUN_DIR"
else
  log "skip (tracker-0 not running - start compose stack first)"
  exit 0
fi

if [[ -n "${REDIS_ADDRS:-}" ]]; then
  IFS=',' read -r -a redis_addrs <<< "$REDIS_ADDRS"
  for addr in "${redis_addrs[@]}"; do
    addr="$(echo "$addr" | xargs)"
    if [[ "$addr" == /* || "$addr" == *".sock"* ]]; then
      check "redis $addr" redis_unix_ping "$addr"
    fi
  done
else
  for i in $(seq 0 $((TRACKER_COUNT - 1))); do
    sock="${RUN_DIR}/redis/redis-${i}.sock"
    check "redis shard $i" redis_unix_ping "$sock"
  done
fi

check "postgres main" pg_unix_ready "${RUN_DIR}/postgresql" "$DB_PORT"

if [[ "${UDS_SMOKE_SKIP_CH:-0}" != "1" ]]; then
  check "clickhouse native" ch_unix_query "${RUN_DIR}/clickhouse/native.sock"
fi

check "control http" http_unix_probe "${RUN_DIR}/control/http.sock" "/health"
check "broker health" http_unix_probe "${RUN_DIR}/broker/health.sock" "/healthz"

for i in $(seq 0 $((TRACKER_COUNT - 1))); do
  sock="${RUN_DIR}/tracker/tracker-${i}.sock"
  check "tracker-$i health" tracker_health_probe "$sock"
done

rp_health="${RUN_DIR}/region-proxy/health.sock"
if socket_exists "$rp_health"; then
  check "region-proxy health" http_unix_probe "$rp_health" "/health"
fi

check "broker gnet socket" socket_exists "${RUN_DIR}/broker/gnet.sock"
if socket_exists "${RUN_DIR}/region-proxy/gnet.sock"; then
  check "region-proxy gnet socket" socket_exists "${RUN_DIR}/region-proxy/gnet.sock"
fi

printf 'fault_proof fault=uds_transport_smoke status=%s pass=%d fail=%d run_dir=%s\n' \
  "$([[ "$FAIL" -eq 0 ]] && echo ok || echo fail)" "$PASS" "$FAIL" "$RUN_DIR"

if [[ "$FAIL" -gt 0 ]]; then
  die "$FAIL check(s) failed ($PASS ok)"
fi

log "all $PASS checks passed"
