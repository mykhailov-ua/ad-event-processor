#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
LUA_DIR="${ROOT}/deploy/nginx/lua"

run_lua_test() {
  if command -v luajit > /dev/null 2>&1; then
    luajit "${LUA_DIR}/tests/tarpit_test.lua" "${LUA_DIR}"
  elif docker ps --format '{{.Names}}' 2> /dev/null | grep -q '^espx-nginx-1$'; then
    docker exec espx-nginx-1 /usr/local/openresty/luajit/bin/luajit /etc/nginx/lua/tests/tarpit_test.lua /etc/nginx/lua
  else
    echo "luajit not found and espx-nginx-1 not running; install luajit or start edge stack" >&2
    exit 1
  fi
}

run_lua_test

EDGE_URL="${EDGE_URL:-http://127.0.0.1:8180}"
METRICS_URL="${EDGE_METRICS_URL:-${EDGE_URL}/edge/metrics}"

if ! curl -sf --max-time 2 "${METRICS_URL}" > /dev/null 2>&1; then
  echo "tarpit_test: offline lua ok (edge not reachable for live smoke)"
  exit 0
fi

before="$(curl -sf "${METRICS_URL}" | awk '/^ad_event_processor_edge_tarpit_total /{print $2; exit}')"
before="${before:-0}"

hdrs=()
for i in $(seq 1 100); do
  hdrs+=(-H "X-Tarpit-${i}: v")
done

curl -sf -o /dev/null "${hdrs[@]}" "${EDGE_URL}/health" || true

after="$(curl -sf "${METRICS_URL}" | awk '/^ad_event_processor_edge_tarpit_total /{print $2; exit}')"
after="${after:-0}"

if [[ "${EDGE_TARPIT_ENABLED:-}" == "1" || "${EDGE_TARPIT_ENABLED:-}" == "true" ]]; then
  if awk -v b="$before" -v a="$after" 'BEGIN { exit !(a > b) }'; then
    echo "fault_proof fault=edge_tarpit_triggered tarpit_before=${before} tarpit_after=${after} baseline_ok=true"
  else
    echo "WARN: EDGE_TARPIT_ENABLED but ad_event_processor_edge_tarpit_total did not increase (before=${before} after=${after})" >&2
    exit 1
  fi
else
  echo "tarpit_test: live edge reachable; set EDGE_TARPIT_ENABLED=1 for integration proof"
fi
