#!/usr/bin/env bash
# node_weights_test.sh: offline Lua tests for edge-node-weights weighted pick (M5.2).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
LUA_DIR="${ROOT}/deploy/nginx/lua"

if command -v luajit >/dev/null 2>&1; then
	luajit "${LUA_DIR}/tests/node_weights_test.lua" "${LUA_DIR}"
elif docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^espx-nginx-1$'; then
	docker exec espx-nginx-1 /usr/local/openresty/luajit/bin/luajit /etc/nginx/lua/tests/node_weights_test.lua /etc/nginx/lua
else
	echo "luajit not found and espx-nginx-1 not running; install luajit or start edge stack" >&2
	exit 1
fi
