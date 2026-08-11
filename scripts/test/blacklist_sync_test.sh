#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
LUA_DIR="${ROOT}/deploy/nginx/lua"

if command -v luajit >/dev/null 2>&1; then
	luajit "${LUA_DIR}/tests/blacklist_sync_test.lua" "${LUA_DIR}"
elif docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^espx-nginx-1$'; then
	docker exec espx-nginx-1 /usr/local/openresty/luajit/bin/luajit /etc/nginx/lua/tests/blacklist_sync_test.lua /etc/nginx/lua
else
	echo "blacklist_sync_test: skip (luajit not installed and espx-nginx-1 not running)"
	exit 0
fi

echo "blacklist_sync_test: OK"
