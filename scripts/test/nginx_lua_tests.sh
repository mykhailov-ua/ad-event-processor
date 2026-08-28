#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
LUA_DIR="${ROOT}/deploy/nginx/lua"
MODE="${1:-all}"

run_lua_test() {
  local lua_test="$1"
  if command -v luajit > /dev/null 2>&1; then
    luajit "${LUA_DIR}/tests/${lua_test}" "${LUA_DIR}"
  elif docker ps --format '{{.Names}}' 2> /dev/null | grep -q '^ad-event-processor-nginx-1$'; then
    docker exec ad-event-processor-nginx-1 /usr/local/openresty/luajit/bin/luajit "/etc/nginx/lua/tests/${lua_test}" /etc/nginx/lua
  else
    echo "nginx_lua_tests: skip ${lua_test} (luajit not installed and ad-event-processor-nginx-1 not running)"
    return 2
  fi
}

run_tarpit_live_smoke() {
  local EDGE_URL="${EDGE_URL:-http://127.0.0.1:8180}"
  local METRICS_URL="${EDGE_METRICS_URL:-${EDGE_URL}/edge/metrics}"

  if ! curl -sf --max-time 2 "${METRICS_URL}" > /dev/null 2>&1; then
    echo "nginx_lua_tests: tarpit live smoke skipped (edge not reachable)"
    return 0
  fi

  local before after
  before="$(curl -sf "${METRICS_URL}" | awk '/^ad_event_processor_edge_tarpit_total /{print $2; exit}')"
  before="${before:-0}"

  local hdrs=()
  local i
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
      echo "nginx_lua_tests: EDGE_TARPIT_ENABLED but ad_event_processor_edge_tarpit_total did not increase (before=${before} after=${after})" >&2
      exit 1
    fi
  else
    echo "nginx_lua_tests: live edge reachable; set EDGE_TARPIT_ENABLED=1 for integration proof"
  fi
}

run_parse_dfa_tests() {
  local LUA_MOUNT="/etc/nginx/lua"

  run_fault_tests() {
    docker exec ad-event-processor-nginx-1 /usr/local/openresty/luajit/bin/luajit "${LUA_MOUNT}/edge_parse_dfa_fault_test.lua" "${LUA_MOUNT}"
  }

  run_tests() {
    docker exec ad-event-processor-nginx-1 /usr/local/openresty/luajit/bin/luajit -e "
package.path = '${LUA_MOUNT}/?.lua;;'
local dfa = require('edge-parse-dfa')
local uuid = string.char(0x0a,0x0b,0x0c,0x0d,0x0e,0x0f,0x10,0x11,0x12,0x13,0x14,0x15,0x16,0x17,0x18,0x19)
local proto = string.char(0x0a, 16) .. uuid .. string.char(0x12, 5) .. 'click'
local cid, err = dfa.extract_campaign_id(proto,
assert(cid == '0a0b0c0d-0e0f-1011-1213-141516171819' and err == nil, 'proto')
local json = '{\"campaign_id\":\"550e8400-e29b-41d4-a716-446655440000\",\"type\":\"click\"}'
cid, err = dfa.extract_campaign_id(json,
assert(cid == '550e8400-e29b-41d4-a716-446655440000' and err == nil, 'json')
assert(dfa.check_content_length(dfa.MAX_BODY_BYTES + 1) == dfa.ERR_OVERSIZE, 'cl')
local over = string.char(0x0a, 100) .. string.rep('a', 100)
cid, err = dfa.extract_campaign_id(over,
assert(err == dfa.ERR_OVERSIZE, 'campaign len')
print('edge-parse-dfa: all tests passed')
"
    run_fault_tests
  }

  if docker ps --format '{{.Names}}' | grep -q '^ad-event-processor-nginx-1$'; then
    run_tests
    return 0
  fi

  if ! command -v docker > /dev/null 2>&1; then
    echo "nginx_lua_tests: parse-dfa skipped (docker unavailable)"
    return 2
  fi

  docker run --rm -v "$ROOT/deploy/nginx/lua:${LUA_MOUNT}:ro" openresty/openresty:alpine \
    /usr/local/openresty/luajit/bin/luajit -e "
package.path = '${LUA_MOUNT}/?.lua;;'
local dfa = require('edge-parse-dfa')
print('edge-parse-dfa: smoke ok')
"
  docker run --rm -v "$ROOT/deploy/nginx/lua:${LUA_MOUNT}:ro" openresty/openresty:alpine \
    /usr/local/openresty/luajit/bin/luajit "${LUA_MOUNT}/edge_parse_dfa_fault_test.lua" "${LUA_MOUNT}"
}

case "$MODE" in
  parse-dfa)
    if ! run_parse_dfa_tests; then
      rc=$?
      if [[ "$rc" -eq 2 ]]; then
        exit 0
      fi
      exit "$rc"
    fi
    ;;
  compliance)
    run_lua_test tarpit_test.lua
    run_lua_test blacklist_sync_test.lua
    ;;
  unit)
    skipped=0
    for lua_test in tarpit_test.lua blacklist_sync_test.lua node_weights_test.lua edge_net_test.lua tls_alpn_test.lua; do
      if ! run_lua_test "$lua_test"; then
        rc=$?
        if [[ "$rc" -eq 2 ]]; then
          skipped=$((skipped + 1))
        else
          exit "$rc"
        fi
      fi
    done
    if [[ "$skipped" -eq 3 ]]; then
      echo "nginx_lua_tests: skip (luajit not installed and ad-event-processor-nginx-1 not running)"
      exit 0
    fi
    ;;
  all)
    run_lua_test tarpit_test.lua
    run_lua_test blacklist_sync_test.lua
    run_lua_test node_weights_test.lua
    run_lua_test edge_net_test.lua
    run_lua_test tls_alpn_test.lua
    run_tarpit_live_smoke
    ;;
  *)
    echo "usage: $0 [all|unit|compliance|parse-dfa]" >&2
    exit 2
    ;;
esac

echo "nginx_lua_tests: OK"
